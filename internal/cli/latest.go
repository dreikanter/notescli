package cli

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/dreikanter/notescli/note"
	"github.com/spf13/cobra"
)

var latestCmd = &cobra.Command{
	Use:   "latest",
	Short: "Print absolute path to the most recent note, optionally filtered by type, slug, or tag",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		root := mustNotesPath()
		n, err := scanAndFilter(cmd, root)
		if err != nil {
			return err
		}

		fmt.Fprintln(cmd.OutOrStdout(), filepath.Join(root, n.RelPath))
		return nil
	},
}

// scanAndFilter scans notes and applies --today, --type, --slug, --tag filter flags,
// returning the most recent match.
func scanAndFilter(cmd *cobra.Command, root string) (*note.Note, error) {
	notes, err := note.Scan(root)
	if err != nil {
		return nil, err
	}

	today, _ := cmd.Flags().GetBool("today")
	types, _ := cmd.Flags().GetStringSlice("type")
	slugs, _ := cmd.Flags().GetStringSlice("slug")
	tags, _ := cmd.Flags().GetStringSlice("tag")

	if today {
		notes = note.FilterByDate(notes, time.Now().Format("20060102"))
	}

	if len(types) > 0 {
		notes = note.FilterByTypes(notes, types)
	}

	if len(slugs) > 0 {
		notes = note.FilterBySlugs(notes, slugs)
	}

	if len(tags) > 0 {
		notes, err = note.FilterByTags(notes, root, tags)
		if err != nil {
			return nil, err
		}
	}

	if len(notes) == 0 {
		if len(types) > 0 || len(slugs) > 0 || len(tags) > 0 || today {
			return nil, fmt.Errorf("no notes found matching the given criteria")
		}
		return nil, fmt.Errorf("no notes found")
	}

	return &notes[0], nil
}

func init() {
	latestCmd.Flags().StringSlice("type", nil, "filter by note type (repeatable)")
	latestCmd.Flags().StringSlice("slug", nil, "filter by slug (repeatable)")
	latestCmd.Flags().StringSlice("tag", nil, "filter by tag (repeatable, all must match)")
	latestCmd.Flags().Bool("today", false, "filter to notes created today")
	rootCmd.AddCommand(latestCmd)
}
