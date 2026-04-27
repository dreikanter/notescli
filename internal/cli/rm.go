package cli

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/dreikanter/notesctl/note"
	"github.com/spf13/cobra"
)

var rmCmd = &cobra.Command{
	Use:   "rm <id>",
	Short: "Delete a note",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("id must be an integer: %s", args[0])
		}

		store, err := notesStore()
		if err != nil {
			return err
		}

		entry, err := store.Get(id)
		if err != nil {
			if errors.Is(err, note.ErrNotFound) {
				return fmt.Errorf("note %d not found", id)
			}
			return err
		}
		path := store.AbsPath(entry)

		if err := store.Delete(id); err != nil {
			return err
		}

		fmt.Fprintln(cmd.OutOrStdout(), path)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(rmCmd)
}
