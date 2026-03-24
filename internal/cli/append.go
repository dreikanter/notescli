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
	Short: "Append text from stdin to a note, optionally creating it",
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

		noteType, _ := cmd.Flags().GetString("type")
		slug, _ := cmd.Flags().GetString("slug")
		tags, _ := cmd.Flags().GetStringSlice("tag")
		create, _ := cmd.Flags().GetBool("create")
		title, _ := cmd.Flags().GetString("title")
		description, _ := cmd.Flags().GetString("description")

		hasFilters := noteType != "" || slug != "" || len(tags) > 0

		if !create {
			if title != "" {
				return fmt.Errorf("--title requires --create")
			}
			if description != "" {
				return fmt.Errorf("--description requires --create")
			}
		}

		if create && len(args) == 1 {
			return fmt.Errorf("--create cannot be combined with positional argument")
		}

		if noteType != "" && !note.IsKnownType(noteType) {
			return fmt.Errorf("unknown note type %q (valid types: %s)", noteType, strings.Join(note.KnownTypes, ", "))
		}

		var targetPath string

		if len(args) == 1 {
			if hasFilters {
				return fmt.Errorf("cannot combine positional argument with filter flags")
			}

			if strings.Contains(args[0], "/") {
				targetPath, err = resolveFilePath(args[0], root)
				if err != nil {
					return err
				}
			} else {
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
			notes, scanErr := note.Scan(root)
			if scanErr != nil {
				return scanErr
			}

			if noteType != "" {
				notes = note.FilterByTypes(notes, []string{noteType})
			}
			if slug != "" {
				notes = note.FilterBySlugs(notes, []string{slug})
			}
			if len(tags) > 0 {
				notes, err = note.FilterByTags(notes, root, tags)
				if err != nil {
					return err
				}
			}

			if len(notes) > 0 {
				targetPath = filepath.Join(root, notes[0].RelPath)
			} else if create {
				targetPath, err = createNote(createNoteParams{
					Root:        root,
					Slug:        slug,
					Type:        noteType,
					Tags:        tags,
					Title:       title,
					Description: description,
				})
				if err != nil {
					return err
				}
			} else {
				return fmt.Errorf("no notes found matching the given criteria")
			}
		} else if create {
			return fmt.Errorf("--create requires filter flags (--type, --slug, --tag)")
		} else {
			return fmt.Errorf("specify a note by positional argument or filter flags (--type, --slug, --tag)")
		}

		// Read existing file (may be newly created)
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
	appendCmd.Flags().String("type", "", "filter by note type")
	appendCmd.Flags().String("slug", "", "filter by slug")
	appendCmd.Flags().StringSlice("tag", nil, "filter by tag (repeatable, all must match)")
	appendCmd.Flags().Bool("create", false, "create note if no match found")
	appendCmd.Flags().String("title", "", "title for frontmatter (requires --create)")
	appendCmd.Flags().String("description", "", "description for frontmatter (requires --create)")
	rootCmd.AddCommand(appendCmd)
}
