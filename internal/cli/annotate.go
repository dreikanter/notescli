package cli

// Claude CLI envelope from `claude -p --output-format json --json-schema ...`:
//
//	{
//	  "type": "result",
//	  "subtype": "success",
//	  "is_error": false,
//	  "result": "<narrative text>",
//	  "structured_output": {<schema-conforming object>},
//	  "session_id": "...",
//	  "duration_ms": 0,
//	  "total_cost_usd": 0
//	}
//
// parseAnnotation reads the outer envelope and pulls the schema-validated
// payload from structured_output. The result field holds Claude's narrative
// response and is used only to surface error messages when is_error is true.

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/dreikanter/notes-cli/note"
	"github.com/spf13/cobra"
)

// claudeBinary is the name or absolute path of the Claude Code CLI binary.
// Tests override this to point at a fake shell script.
var claudeBinary = "claude"

const annotateDefaultModel = "claude-haiku-4-5"

const annotateSystemPrompt = `You are annotating a personal note stored as a markdown file.
Generate concise metadata for the provided note body, returning ONLY the fields required by the response schema.
- title: short title, <= 8 words.
- description: single-sentence summary, <= 140 characters.
- tags: 1-3 lowercase single-word slugs related to the content.`

var annotateCmd = &cobra.Command{
	Use:   "annotate <id|type|query>",
	Short: "Fill empty frontmatter (title, description, tags) using Claude Code CLI",
	Args:  cobra.ExactArgs(1),
	RunE:  annotateRunE,
}

func annotateRunE(cmd *cobra.Command, args []string) error {
	model, _ := cmd.Flags().GetString("model")
	maxChars, _ := cmd.Flags().GetInt("max-chars")

	root := mustNotesPath()
	n, err := note.ResolveRef(root, args[0])
	if err != nil {
		return err
	}

	fullPath := filepath.Join(root, n.RelPath)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("cannot read note: %w", err)
	}

	existing := note.ParseFrontmatterFields(data)
	body := note.StripFrontmatter(data)

	empty := annotateEmptyFields(existing)
	if len(empty) == 0 {
		fmt.Fprintln(cmd.OutOrStdout(), fullPath)
		return nil
	}

	if len(bytes.TrimSpace(body)) == 0 {
		return errors.New("note has no body content to annotate")
	}

	prompt := string(body)
	if maxChars > 0 {
		if runes := []rune(prompt); len(runes) > maxChars {
			prompt = string(runes[:maxChars])
			fmt.Fprintf(cmd.ErrOrStderr(), "truncated note body to %d chars for annotation\n", maxChars)
		}
	}

	schema := buildAnnotateSchema(empty)
	out, err := runClaude(model, schema, prompt)
	if err != nil {
		return err
	}

	gen, err := parseAnnotation(out)
	if err != nil {
		return err
	}

	merged := mergeAnnotation(existing, gen)
	newContent := note.BuildFrontmatter(merged) + string(body)

	tmpPath := fullPath + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(newContent), 0o644); err != nil {
		return fmt.Errorf("cannot write note: %w", err)
	}
	if err := os.Rename(tmpPath, fullPath); err != nil {
		os.Remove(tmpPath)
		return fmt.Errorf("cannot rename note: %w", err)
	}

	fmt.Fprintln(cmd.OutOrStdout(), fullPath)
	return nil
}

// runClaude executes the Claude Code CLI non-interactively and returns its stdout.
// Returns a clear error if the binary is not found or exits non-zero.
func runClaude(model, schema, prompt string) ([]byte, error) {
	bin, err := exec.LookPath(claudeBinary)
	if err != nil {
		return nil, errors.New("claude CLI not found in PATH")
	}

	args := []string{
		"-p",
		"--model", model,
		"--output-format", "json",
		"--json-schema", schema,
		"--append-system-prompt", annotateSystemPrompt,
		prompt,
	}

	c := exec.Command(bin, args...)
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr
	if err := c.Run(); err != nil {
		msg := stderr.String()
		if msg == "" {
			msg = err.Error()
		}
		return nil, fmt.Errorf("claude failed: %s", msg)
	}
	return stdout.Bytes(), nil
}

// annotateEmptyFields returns the empty fields among {title, description, tags}
// in a deterministic order. "tags" counts as empty when the slice is empty.
func annotateEmptyFields(f note.FrontmatterFields) []string {
	var empty []string
	if f.Title == "" {
		empty = append(empty, "title")
	}
	if f.Description == "" {
		empty = append(empty, "description")
	}
	if len(f.Tags) == 0 {
		empty = append(empty, "tags")
	}
	return empty
}

// buildAnnotateSchema returns a JSON Schema string requiring only the given fields.
// Fields must be a subset of {"title", "description", "tags"}.
func buildAnnotateSchema(fields []string) string {
	props := map[string]any{}
	for _, f := range fields {
		switch f {
		case "title", "description":
			props[f] = map[string]string{"type": "string"}
		case "tags":
			props[f] = map[string]any{
				"type":     "array",
				"items":    map[string]string{"type": "string"},
				"maxItems": 3,
			}
		}
	}
	schema := map[string]any{
		"type":                 "object",
		"properties":           props,
		"required":             fields,
		"additionalProperties": false,
	}
	b, _ := json.Marshal(schema)
	return string(b)
}

// annotateEnvelope mirrors the outer JSON written by `claude -p --output-format json`.
// Only the fields we rely on are declared.
type annotateEnvelope struct {
	IsError          bool            `json:"is_error"`
	Result           string          `json:"result"`
	StructuredOutput *annotateResult `json:"structured_output"`
}

// annotateResult is the schema-validated payload carried by annotateEnvelope.StructuredOutput.
type annotateResult struct {
	Title       string   `json:"title,omitempty"`
	Description string   `json:"description,omitempty"`
	Tags        []string `json:"tags,omitempty"`
}

// parseAnnotation unmarshals the claude CLI stdout into an annotateResult.
func parseAnnotation(raw []byte) (annotateResult, error) {
	var env annotateEnvelope
	if err := json.Unmarshal(raw, &env); err != nil {
		return annotateResult{}, fmt.Errorf("cannot parse claude response: %w", err)
	}
	if env.IsError {
		return annotateResult{}, fmt.Errorf("claude returned error: %s", env.Result)
	}
	if env.StructuredOutput == nil {
		return annotateResult{}, fmt.Errorf("claude response missing structured_output; got result: %s", snippet(env.Result, 200))
	}
	return *env.StructuredOutput, nil
}

// snippet returns up to n bytes of s, with "..." appended when truncated.
func snippet(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}

// mergeAnnotation fills empty fields in existing from gen.
// Non-empty fields in existing are preserved.
func mergeAnnotation(existing note.FrontmatterFields, gen annotateResult) note.FrontmatterFields {
	merged := existing
	if merged.Title == "" {
		merged.Title = gen.Title
	}
	if merged.Description == "" {
		merged.Description = gen.Description
	}
	if len(merged.Tags) == 0 && len(gen.Tags) > 0 {
		merged.Tags = gen.Tags
	}
	return merged
}

func init() {
	annotateCmd.Flags().String("model", annotateDefaultModel, "Claude model to use")
	annotateCmd.Flags().Int("max-chars", 0, "truncate note body to this many characters before annotating (0 = no limit)")
	rootCmd.AddCommand(annotateCmd)
}
