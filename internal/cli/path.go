package cli

import (
	"fmt"

	"github.com/spf13/cobra"
)

var pathCmd = &cobra.Command{
	Use:   "path",
	Short: "Print the notes store path",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(cmd.OutOrStdout(), mustNotesPath())
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pathCmd)
}
