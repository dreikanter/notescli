package cli

import (
	"errors"
	"fmt"
	"os"
	"strconv"

	"github.com/dreikanter/notesctl/note"
	"github.com/spf13/cobra"
)

var readCmd = &cobra.Command{
	Use:   "read <id>",
	Short: "Read a note",
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

		data, err := os.ReadFile(store.AbsPath(entry))
		if err != nil {
			return err
		}

		noFrontmatter, _ := cmd.Flags().GetBool("no-frontmatter")
		if noFrontmatter {
			data = note.StripFrontmatter(data)
		}

		_, err = cmd.OutOrStdout().Write(data)
		return err
	},
}

func registerReadFlags() {
	readCmd.Flags().Bool("no-frontmatter", false, "exclude YAML frontmatter from output")
}

func init() {
	registerReadFlags()
	rootCmd.AddCommand(readCmd)
}
