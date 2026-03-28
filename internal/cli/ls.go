package cli

import (
	"fmt"
	"time"

	"github.com/dreikanter/notescli/note"
	"github.com/spf13/cobra"
)

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List recent notes",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		lsLimit, _ := cmd.Flags().GetInt("limit")
		lsType, _ := cmd.Flags().GetString("type")
		lsSlug, _ := cmd.Flags().GetString("slug")
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

		if lsType != "" {
			notes = note.FilterByType(notes, lsType)
		}

		if lsSlug != "" {
			notes = note.FilterBySlug(notes, lsSlug)
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
			fmt.Fprintln(cmd.OutOrStdout(), n.RelPath)
		}
		return nil
	},
}

func init() {
	lsCmd.Flags().Int("limit", 20, "maximum number of notes to list")
	lsCmd.Flags().String("type", "", "filter by note type, e.g. todo, backlog, weekly")
	lsCmd.Flags().String("slug", "", "filter by descriptive slug")
	lsCmd.Flags().StringSlice("tag", nil, "filter by frontmatter tag (repeatable, AND logic)")
	lsCmd.Flags().String("name", "", "filter by filename fragment (case-insensitive substring)")
	lsCmd.Flags().Bool("today", false, "filter notes created today")
	rootCmd.AddCommand(lsCmd)
}
