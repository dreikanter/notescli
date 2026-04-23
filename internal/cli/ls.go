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
		idx, err := note.Load(root, loadOptsFor(cmd, f)...)
		if err != nil {
			return err
		}
		entries := idx.Entries()

		if lsName != "" {
			entries = note.Filter(entries, lsName)
		}

		entries = applyFilters(entries, f)

		if lsLimit > 0 && len(entries) > lsLimit {
			entries = entries[:lsLimit]
		}

		for _, e := range entries {
			fmt.Fprintln(cmd.OutOrStdout(), filepath.Join(root, e.RelPath))
		}
		return nil
	},
}

func registerLsFlags() {
	lsCmd.Flags().Int("limit", 0, "maximum number of notes to list (0 = no limit)")
	lsCmd.Flags().String("name", "", "filter by filename fragment (case-insensitive substring)")
	addFilterFlags(lsCmd)
}

func init() {
	registerLsFlags()
	rootCmd.AddCommand(lsCmd)
}
