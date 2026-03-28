package cli

import (
	"fmt"
	"path/filepath"

	"github.com/dreikanter/notescli/note"
	"github.com/spf13/cobra"
)

var resolveCmd = &cobra.Command{
	Use:   "resolve <id|path|basename|slug|type>",
	Short: "Resolve a note reference and print its absolute path",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root := mustNotesPath()
		n, err := note.ResolveRef(root, args[0])
		if err != nil {
			return err
		}

		fmt.Fprintln(cmd.OutOrStdout(), filepath.Join(root, n.RelPath))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(resolveCmd)
}
