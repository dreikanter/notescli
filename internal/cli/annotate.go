package cli

// Claude CLI envelope (UNVERIFIED — run-time probe blocked by sandbox,
// shape derived from claude -p --output-format json docs):
//
//	{
//	  "type": "result",
//	  "subtype": "success",
//	  "is_error": false,
//	  "result": "<schema-conforming JSON string>",
//	  "session_id": "...",
//	  "duration_ms": 0,
//	  "total_cost_usd": 0
//	}
//
// The schema-validated payload is the result field (as a JSON string).
// Task 4 tests must be verified against a real invocation before merging.
// If the observed shape differs on another machine, update annotateEnvelope
// and parseAnnotation in Task 4 accordingly.

import (
	"encoding/json"
	"errors"
	"fmt"

	"github.com/dreikanter/notes-cli/note"
	"github.com/spf13/cobra"
)

// claudeBinary is the name or absolute path of the Claude Code CLI binary.
// Tests override this to point at a fake shell script.
var claudeBinary = "claude"

const annotateDefaultModel = "claude-haiku-4-5"

var annotateCmd = &cobra.Command{
	Use:   "annotate <id|type|query>",
	Short: "Fill empty frontmatter (title, description, tags) using Claude Code CLI",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return errors.New("not implemented")
	},
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
				"type":  "array",
				"items": map[string]string{"type": "string"},
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
	IsError bool   `json:"is_error"`
	Result  string `json:"result"`
}

// annotateResult is the schema-validated payload carried by annotateEnvelope.Result.
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
	var res annotateResult
	if err := json.Unmarshal([]byte(env.Result), &res); err != nil {
		return annotateResult{}, fmt.Errorf("cannot parse claude response payload: %w", err)
	}
	return res, nil
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
	rootCmd.AddCommand(annotateCmd)
}
