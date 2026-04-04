package cli

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/dreikanter/notescli/note"
	"github.com/spf13/cobra"
)

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List notes",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		lsLimit, _ := cmd.Flags().GetInt("limit")
		lsTypes, _ := cmd.Flags().GetStringSlice("type")
		lsSlugs, _ := cmd.Flags().GetStringSlice("slug")
		lsTags, _ := cmd.Flags().GetStringSlice("tag")
		lsName, _ := cmd.Flags().GetString("name")
		lsToday, _ := cmd.Flags().GetBool("today")

		root := mustNotesPath()
		notes, err := note.Scan(root)
		if err != nil {
			return err
		}

		if lsToday {
			notes = note.FilterByDate(notes, time.Now().Format("20060102"))
		}

		if lsName != "" {
			notes = note.Filter(notes, lsName)
		}

		if len(lsTypes) > 0 {
			notes = note.FilterByTypes(notes, lsTypes)
		}

		if len(lsSlugs) > 0 {
			notes = note.FilterBySlugs(notes, lsSlugs)
		}

		if len(lsTags) > 0 {
			notes, err = note.FilterByTags(notes, root, lsTags)
			if err != nil {
				return err
			}
		}

		if lsLimit > 0 && len(notes) > lsLimit {
			notes = notes[:lsLimit]
		}

		for _, n := range notes {
			fmt.Fprintln(cmd.OutOrStdout(), filepath.Join(root, n.RelPath))
		}
		return nil
	},
}

func init() {
	lsCmd.Flags().Int("limit", 0, "maximum number of notes to list (0 = no limit)")
	lsCmd.Flags().StringSlice("type", nil, "filter by note type (repeatable)")
	lsCmd.Flags().StringSlice("slug", nil, "filter by descriptive slug (repeatable)")
	lsCmd.Flags().StringSlice("tag", nil, "filter by frontmatter tag (repeatable, AND logic)")
	lsCmd.Flags().String("name", "", "filter by filename fragment (case-insensitive substring)")
	lsCmd.Flags().Bool("today", false, "filter notes created today")
	rootCmd.AddCommand(lsCmd)
}
