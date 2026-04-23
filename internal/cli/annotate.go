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
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"time"

	"github.com/dreikanter/notes-cli/note"
	"github.com/spf13/cobra"
)

// claudeBinary is the name or absolute path of the Claude Code CLI binary.
// Tests override this to point at a fake shell script.
var claudeBinary = "claude"

const annotateDefaultModel = "claude-haiku-4-5"

const annotateDefaultTimeout = 60 * time.Second

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
	timeout, _ := cmd.Flags().GetDuration("timeout")

	root, err := notesRoot()
	if err != nil {
		return err
	}
	n, err := resolveRef(cmd, root, args[0])
	if err != nil {
		return err
	}

	fullPath := filepath.Join(root, n.RelPath)
	data, err := os.ReadFile(fullPath)
	if err != nil {
		return fmt.Errorf("cannot read note: %w", err)
	}

	existing, body, err := note.ParseNote(data)
	if err != nil {
		return fmt.Errorf("%s: %w", fullPath, err)
	}

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

	schema, err := buildAnnotateSchema(empty)
	if err != nil {
		return err
	}
	ctx := cmd.Context()
	if timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}
	out, err := runClaude(ctx, model, schema, prompt)
	if err != nil {
		return err
	}

	gen, err := parseAnnotation(out)
	if err != nil {
		return err
	}

	merged := mergeAnnotation(existing, gen)
	newContent, err := note.FormatNote(merged, body)
	if err != nil {
		return err
	}

	if err := writeAtomic(fullPath, newContent); err != nil {
		return err
	}

	fmt.Fprintln(cmd.OutOrStdout(), fullPath)
	return nil
}

// runClaude executes the Claude Code CLI non-interactively and returns its stdout.
// Returns a clear error if the binary is not found, the context is cancelled
// (e.g. timeout), or the child exits non-zero.
func runClaude(ctx context.Context, model, schema, prompt string) ([]byte, error) {
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

	c := exec.CommandContext(ctx, bin, args...)
	var stdout, stderr bytes.Buffer
	c.Stdout = &stdout
	c.Stderr = &stderr
	if err := c.Run(); err != nil {
		if errors.Is(ctx.Err(), context.DeadlineExceeded) {
			return nil, errors.New("claude timed out; pass --timeout to raise the limit")
		}
		if s := stderr.String(); s != "" {
			return nil, fmt.Errorf("claude failed: %s", s)
		}
		exitCode := -1
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			exitCode = ee.ExitCode()
		}
		if out := stdout.String(); out != "" {
			return nil, fmt.Errorf("claude failed (exit %d): %s", exitCode, snippet(out, 500))
		}
		return nil, fmt.Errorf("claude failed (exit %d): %s", exitCode, err.Error())
	}
	return stdout.Bytes(), nil
}

// annotateEmptyFields returns the empty fields among {title, description, tags}
// in a deterministic order. "tags" counts as empty when the slice is empty.
func annotateEmptyFields(f note.Frontmatter) []string {
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
func buildAnnotateSchema(fields []string) (string, error) {
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
	b, err := json.Marshal(schema)
	if err != nil {
		return "", fmt.Errorf("cannot marshal annotate schema: %w", err)
	}
	return string(b), nil
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
func mergeAnnotation(existing note.Frontmatter, gen annotateResult) note.Frontmatter {
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

func registerAnnotateFlags() {
	annotateCmd.Flags().String("model", annotateDefaultModel, "Claude model to use")
	annotateCmd.Flags().Int("max-chars", 0, "truncate note body to this many characters before annotating (0 = no limit)")
	annotateCmd.Flags().Duration("timeout", annotateDefaultTimeout, "maximum time to wait for the claude CLI to respond (0 = no timeout)")
}

func init() {
	registerAnnotateFlags()
	rootCmd.AddCommand(annotateCmd)
}
