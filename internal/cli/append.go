package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dreikanter/notescli/note"
	"github.com/spf13/cobra"
)

var appendCmd = &cobra.Command{
	Use:   "append [<id|path|basename|slug|type>]",
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

		todayDate := time.Now().Format("20060102")

		noteType, _ := cmd.Flags().GetString("type")
		slug, _ := cmd.Flags().GetString("slug")
		tags, _ := cmd.Flags().GetStringSlice("tag")
		create, _ := cmd.Flags().GetBool("create")
		today, _ := cmd.Flags().GetBool("today")
		title, _ := cmd.Flags().GetString("title")
		description, _ := cmd.Flags().GetString("description")

		hasFilters := noteType != "" || slug != "" || len(tags) > 0
		canCreate := create || today

		if !canCreate {
			if title != "" {
				return fmt.Errorf("--title requires --create or --today")
			}
			if description != "" {
				return fmt.Errorf("--description requires --create or --today")
			}
		}

		if create && today {
			return fmt.Errorf("--create and --today are mutually exclusive")
		}

		flagName := "create"
		if today {
			flagName = "today"
		}

		if canCreate && len(args) == 1 {
			return fmt.Errorf("--%s cannot be combined with positional argument", flagName)
		}

		if noteType != "" && !note.IsKnownType(noteType) {
			return fmt.Errorf("unknown note type %q (valid types: %s)", noteType, strings.Join(note.KnownTypes, ", "))
		}

		var targetPath string

		if len(args) == 1 {
			if hasFilters {
				return fmt.Errorf("cannot combine positional argument with filter flags")
			}

			n, resolveErr := note.ResolveRef(root, args[0])
			if resolveErr != nil {
				return resolveErr
			}
			targetPath = filepath.Join(root, n.RelPath)
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

			needsCreate := false
			if len(notes) > 0 {
				if today && notes[0].Date != todayDate {
					needsCreate = true
				} else {
					targetPath = filepath.Join(root, notes[0].RelPath)
				}
			} else if canCreate {
				needsCreate = true
			} else {
				return fmt.Errorf("no notes found matching the given criteria")
			}

			if !needsCreate && (title != "" || description != "") {
				fmt.Fprintln(cmd.ErrOrStderr(), "warning: --title and --description are ignored when appending to an existing note")
			}

			if needsCreate {
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
			}
		} else if canCreate {
			return fmt.Errorf("--%s requires filter flags (--type, --slug, --tag)", flagName)
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

func init() {
	appendCmd.Flags().String("type", "", "filter by note type")
	appendCmd.Flags().String("slug", "", "filter by slug")
	appendCmd.Flags().StringSlice("tag", nil, "filter by tag (repeatable, all must match)")
	appendCmd.Flags().Bool("create", false, "create note if no match found")
	appendCmd.Flags().Bool("today", false, "append to today's note or create a new one")
	appendCmd.Flags().String("title", "", "title for frontmatter (requires --create or --today)")
	appendCmd.Flags().String("description", "", "description for frontmatter (requires --create or --today)")
	rootCmd.AddCommand(appendCmd)
}
