package cli

import "github.com/spf13/cobra"

var rgCmd = &cobra.Command{
	Use:   "rg [flags] <pattern>",
	Short: "Search note contents using ripgrep",
	Long: `Search note contents using ripgrep (rg). Only .md files are searched.

The following flag is injected automatically: --glob *.md. The notes path is appended as the last argument. Pass any other rg flags explicitly (e.g. --ignore-case, --heading, --sortr path).`,
	DisableFlagParsing: true,
	SilenceErrors:      true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runExternalSearch(
			cmd, args, "rg",
			"ripgrep (rg) is not installed; install it from https://github.com/BurntSushi/ripgrep",
			func(root string, passThrough []string) []string {
				return append(append([]string{"--glob", "*.md"}, passThrough...), root)
			},
		)
	},
}

func init() {
	rootCmd.AddCommand(rgCmd)
}
