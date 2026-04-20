package cli

import (
	"fmt"

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
		tags, err := note.ExtractTags(root)
		if err != nil {
			return err
		}
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
