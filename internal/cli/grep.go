package cli

import "github.com/spf13/cobra"

var grepCmd = &cobra.Command{
	Use:   "grep [flags] <pattern>",
	Short: "Search note contents using grep",
	Long: `Search note contents using grep. Only .md files are searched; .git directories are excluded.

The following flags are injected automatically: -r (recursive), --include=*.md, --exclude-dir=.git. The notes path is appended as the last argument. Pass -i explicitly for case-insensitive search.`,
	DisableFlagParsing: true,
	SilenceErrors:      true,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runExternalSearch(cmd, args, "grep", "", func(root string, passThrough []string) []string {
			return append(append([]string{"-r", "--include=*.md", "--exclude-dir=.git"}, passThrough...), root)
		})
	},
}

func init() {
	rootCmd.AddCommand(grepCmd)
}
