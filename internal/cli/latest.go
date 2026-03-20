package cli

import (
	"fmt"
	"path/filepath"

	"github.com/dreikanter/notescli/note"
	"github.com/spf13/cobra"
)

var latestCmd = &cobra.Command{
	Use:   "latest [type]",
	Short: "Print absolute path to the most recent note, optionally filtered by type",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root := mustNotesPath()
		notes, err := note.Scan(root)
		if err != nil {
			return err
		}

		if len(args) > 0 {
			notes = note.FilterBySlug(notes, args[0])
		}

		if len(notes) == 0 {
			if len(args) > 0 {
				return fmt.Errorf("no notes found with type %q", args[0])
			}
			return fmt.Errorf("no notes found")
		}

		fmt.Fprintln(cmd.OutOrStdout(), filepath.Join(root, notes[0].RelPath))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(latestCmd)
}
