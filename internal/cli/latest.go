package cli

import (
	"fmt"
	"path/filepath"

	"github.com/dreikanter/notescli/note"
	"github.com/spf13/cobra"
)

var latestCmd = &cobra.Command{
	Use:   "latest",
	Short: "Print absolute path to the most recent note, optionally filtered by type, slug, or tag",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		root := mustNotesPath()
		notes, err := note.Scan(root)
		if err != nil {
			return err
		}

		types, _ := cmd.Flags().GetStringSlice("type")
		slugs, _ := cmd.Flags().GetStringSlice("slug")
		tags, _ := cmd.Flags().GetStringSlice("tag")

		if len(types) > 0 {
			notes = note.FilterByTypes(notes, types)
		}

		if len(slugs) > 0 {
			notes = note.FilterBySlugs(notes, slugs)
		}

		if len(tags) > 0 {
			notes, err = note.FilterByTags(notes, root, tags)
			if err != nil {
				return err
			}
		}

		if len(notes) == 0 {
			if len(types) > 0 || len(slugs) > 0 || len(tags) > 0 {
				return fmt.Errorf("no notes found matching the given criteria")
			}
			return fmt.Errorf("no notes found")
		}

		fmt.Fprintln(cmd.OutOrStdout(), filepath.Join(root, notes[0].RelPath))
		return nil
	},
}

func init() {
	latestCmd.Flags().StringSlice("type", nil, "filter by note type (repeatable)")
	latestCmd.Flags().StringSlice("slug", nil, "filter by slug (repeatable)")
	latestCmd.Flags().StringSlice("tag", nil, "filter by tag (repeatable, all must match)")
	rootCmd.AddCommand(latestCmd)
}
