package cli

import "github.com/spf13/cobra"

var pathCmd = &cobra.Command{
	Use:   "path",
	Short: "Print the notes archive path",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		cmd.Println(mustNotesPath())
		return nil
	},
}

func init() {
	rootCmd.AddCommand(pathCmd)
}
