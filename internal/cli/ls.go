package cli

import (
	"fmt"

	"github.com/dreikanter/notescli/note"
	"github.com/spf13/cobra"
)

var (
	lsLimit int
	lsType  string
	lsTags  []string
)

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List recent notes",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		root := mustNotesPath()
		notes, err := note.Scan(root)
		if err != nil {
			return err
		}

		if lsType != "" {
			notes = note.FilterBySlug(notes, lsType)
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
	lsCmd.Flags().IntVar(&lsLimit, "limit", 20, "maximum number of notes to list")
	lsCmd.Flags().StringVar(&lsType, "type", "", "filter by note type (slug), e.g. todo, backlog, weekly")
	lsCmd.Flags().StringSliceVar(&lsTags, "tag", nil, "filter by frontmatter tag (repeatable, AND logic)")
	rootCmd.AddCommand(lsCmd)
}
