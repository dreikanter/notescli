package cli

import (
	"fmt"
	"sort"

	"github.com/spf13/cobra"
)

var tagsCmd = &cobra.Command{
	Use:   "tags",
	Short: "List all tags from frontmatter and body hashtags",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := notesStore()
		if err != nil {
			return err
		}
		entries, err := store.All()
		if err != nil {
			return err
		}
		set := make(map[string]struct{})
		for _, e := range entries {
			for _, t := range e.Meta.Tags {
				set[t] = struct{}{}
			}
		}
		tags := make([]string, 0, len(set))
		for t := range set {
			tags = append(tags, t)
		}
		sort.Strings(tags)
		out := cmd.OutOrStdout()
		for _, t := range tags {
			fmt.Fprintln(out, t)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(tagsCmd)
}
