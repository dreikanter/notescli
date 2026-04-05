package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dreikanter/notescli/note"
	"github.com/spf13/cobra"
)

var readCmd = &cobra.Command{
	Use:   "read [<id|type|query>]",
	Short: "Read a note by ref or filter flags",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root := mustNotesPath()

		noteType, _ := cmd.Flags().GetString("type")
		slug, _ := cmd.Flags().GetString("slug")
		tags, _ := cmd.Flags().GetStringSlice("tag")
		today, _ := cmd.Flags().GetBool("today")
		noFrontmatter, _ := cmd.Flags().GetBool("no-frontmatter")

		hasFilters := noteType != "" || slug != "" || len(tags) > 0 || today

		var relPath string

		if len(args) == 1 {
			if hasFilters {
				return fmt.Errorf("cannot combine positional argument with filter flags")
			}
			n, err := note.ResolveRef(root, args[0])
			if err != nil {
				return err
			}
			relPath = n.RelPath
		} else if hasFilters {
			notes, err := note.Scan(root)
			if err != nil {
				return err
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
			relPath = notes[0].RelPath
		} else {
			return fmt.Errorf("specify a note by positional argument or filter flags (--type, --slug, --tag, --today)")
		}

		data, err := os.ReadFile(filepath.Join(root, relPath))
		if err != nil {
			return err
		}

		if noFrontmatter {
			data = note.StripFrontmatter(data)
		}

		_, err = cmd.OutOrStdout().Write(data)
		return err
	},
}

func registerReadFlags() {
	readCmd.Flags().String("type", "", "filter by note type")
	readCmd.Flags().String("slug", "", "filter by slug")
	readCmd.Flags().StringSlice("tag", nil, "filter by tag (repeatable, all must match)")
	readCmd.Flags().Bool("today", false, "only match notes created today")
	readCmd.Flags().BoolP("no-frontmatter", "F", false, "exclude YAML frontmatter from output")
}

func init() {
	registerReadFlags()
	rootCmd.AddCommand(readCmd)
}
