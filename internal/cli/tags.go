package cli

import (
	"fmt"
	"sort"

	"github.com/dreikanter/notes-cli/note"
	"github.com/spf13/cobra"
)

var tagsCmd = &cobra.Command{
	Use:   "tags",
	Short: "List all tags from frontmatter and body hashtags",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := notesRoot()
		if err != nil {
			return err
		}
		idx, err := note.Load(root, note.WithLogger(stderrLogger(cmd)))
		if err != nil {
			return err
		}
		set := make(map[string]struct{})
		for _, t := range idx.Tags() {
			set[t] = struct{}{}
		}
		for _, e := range idx.Entries() {
			for _, t := range e.BodyHashtags() {
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
