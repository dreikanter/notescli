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
	Use:   "append [<id|type|query>]",
	Short: "Append text from stdin to a note",
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
		today, _ := cmd.Flags().GetBool("today")

		hasFilters := noteType != "" || slug != "" || len(tags) > 0 || today

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

			if today {
				notes = note.FilterByDate(notes, time.Now().Format("20060102"))
			}
			if noteType != "" {
				notes = note.FilterByTypes(notes, []string{noteType})
			}
			if slug != "" {
				notes = note.FilterBySlug(notes, slug)
			}
			if len(tags) > 0 {
				notes, err = note.FilterByTags(notes, root, tags)
				if err != nil {
					return err
				}
			}

			if len(notes) == 0 {
				return fmt.Errorf("no notes found matching the given criteria")
			}
			targetPath = filepath.Join(root, notes[0].RelPath)
		} else {
			return fmt.Errorf("specify a note by positional argument or filter flags (--type, --slug, --tag, --today)")
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

func registerAppendFlags() {
	appendCmd.Flags().String("type", "", "filter by note type")
	appendCmd.Flags().String("slug", "", "filter by slug")
	appendCmd.Flags().StringSlice("tag", nil, "filter by tag (repeatable, all must match)")
	appendCmd.Flags().Bool("today", false, "only match notes created today")
}

func init() {
	registerAppendFlags()
	rootCmd.AddCommand(appendCmd)
}
