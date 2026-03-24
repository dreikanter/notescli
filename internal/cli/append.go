package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/dreikanter/notescli/note"
	"github.com/spf13/cobra"
)

var appendCmd = &cobra.Command{
	Use:   "append [<id|slug|filename|path>]",
	Short: "Append text from stdin to an existing note",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root := mustNotesPath()

		// Check stdin is piped
		if isTerminal(os.Stdin) {
			return fmt.Errorf("no input: pipe text to stdin (e.g. echo 'text' | notes append <target>)")
		}

		// Read and trim stdin
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("cannot read stdin: %w", err)
		}
		content := strings.TrimSpace(string(data))
		if content == "" {
			return nil
		}

		types, _ := cmd.Flags().GetStringSlice("type")
		slugs, _ := cmd.Flags().GetStringSlice("slug")
		tags, _ := cmd.Flags().GetStringSlice("tag")
		hasFilters := len(types) > 0 || len(slugs) > 0 || len(tags) > 0

		var targetPath string

		if len(args) == 1 {
			if hasFilters {
				return fmt.Errorf("cannot combine positional argument with filter flags")
			}

			if strings.Contains(args[0], "/") {
				// Treat as file path
				targetPath, err = resolveFilePath(args[0], root)
				if err != nil {
					return err
				}
			} else {
				// Resolve by ID/slug/filename
				notes, scanErr := note.Scan(root)
				if scanErr != nil {
					return scanErr
				}
				n := note.Resolve(notes, args[0])
				if n == nil {
					return fmt.Errorf("note not found: %s", args[0])
				}
				targetPath = filepath.Join(root, n.RelPath)
			}
		} else if hasFilters {
			n, filterErr := scanAndFilter(cmd, root)
			if filterErr != nil {
				return filterErr
			}
			targetPath = filepath.Join(root, n.RelPath)
		} else {
			return fmt.Errorf("specify a note by positional argument or filter flags (--type, --slug, --tag)")
		}

		// Read existing file
		existing, err := os.ReadFile(targetPath)
		if err != nil {
			return fmt.Errorf("cannot read note: %w", err)
		}

		// Append: ensure existing ends with \n, then \n + content + \n
		existingStr := string(existing)
		if len(existingStr) > 0 && !strings.HasSuffix(existingStr, "\n") {
			existingStr += "\n"
		}
		result := existingStr + "\n" + content + "\n"

		if err := os.WriteFile(targetPath, []byte(result), 0o644); err != nil {
			return fmt.Errorf("cannot write note: %w", err)
		}

		fmt.Fprintln(cmd.OutOrStdout(), targetPath)
		return nil
	},
}

func resolveFilePath(arg, root string) (string, error) {
	absPath, err := filepath.Abs(arg)
	if err != nil {
		return "", fmt.Errorf("cannot resolve path: %w", err)
	}
	absPath, err = filepath.EvalSymlinks(absPath)
	if err != nil {
		return "", fmt.Errorf("note not found: %s", arg)
	}

	absRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		return "", fmt.Errorf("cannot resolve notes path: %w", err)
	}

	if !strings.HasPrefix(absPath, absRoot+"/") {
		return "", fmt.Errorf("path is outside notes directory: %s", arg)
	}

	return absPath, nil
}

func init() {
	appendCmd.Flags().StringSlice("type", nil, "filter by note type (repeatable)")
	appendCmd.Flags().StringSlice("slug", nil, "filter by slug (repeatable)")
	appendCmd.Flags().StringSlice("tag", nil, "filter by tag (repeatable, all must match)")
	rootCmd.AddCommand(appendCmd)
}
