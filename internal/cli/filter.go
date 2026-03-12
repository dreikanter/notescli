package cli

import (
	"fmt"

	"github.com/dreikanter/notescli/note"
	"github.com/spf13/cobra"
)

var filterCmd = &cobra.Command{
	Use:   "filter <fragment>",
	Short: "Find notes matching a fragment in ID, slug, or filename",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root := mustNotesPath()
		notes, err := note.Scan(root)
		if err != nil {
			return err
		}

		matches := note.Filter(notes, args[0])
		for _, n := range matches {
			fmt.Println(n.RelPath)
		}
		return nil
	},
}

func init() {
	rootCmd.AddCommand(filterCmd)
}
