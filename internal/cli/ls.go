package cli

import (
	"fmt"
	"path/filepath"

	"github.com/dreikanter/notes-cli/note"
	"github.com/spf13/cobra"
)

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List notes",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		lsLimit, _ := cmd.Flags().GetInt("limit")
		lsName, _ := cmd.Flags().GetString("name")
		f := readFilterFlags(cmd)

		root, err := notesRoot()
		if err != nil {
			return err
		}
		notes, err := note.Scan(root)
		if err != nil {
			return err
		}

		if lsName != "" {
			notes = note.Filter(notes, lsName)
		}

		notes, err = applyFilters(notes, root, f)
		if err != nil {
			return err
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
	lsCmd.Flags().String("name", "", "filter by filename fragment (case-insensitive substring)")
	addFilterFlags(lsCmd)
	rootCmd.AddCommand(lsCmd)
}
