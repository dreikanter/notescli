package cli

import (
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var grepCmd = &cobra.Command{
	Use:   "grep [flags] <pattern>",
	Short: "Search note contents using grep",
	Long: `Search note contents using grep. Only .md files are searched; .git directories are excluded.

The following flags are injected automatically: -r (recursive), -i (case-insensitive), --include=*.md, --exclude-dir=.git. The notes path is appended as the last argument.`,
	DisableFlagParsing: true,
	SilenceErrors:      true,
	RunE: func(cmd *cobra.Command, args []string) error {
		for _, arg := range args {
			if arg == "--help" {
				return cmd.Help()
			}
		}

		root := mustNotesPath()
		grepArgs := append([]string{"-r", "-i", "--include=*.md", "--exclude-dir=.git"}, args...)
		grepArgs = append(grepArgs, root)

		grep := exec.Command("grep", grepArgs...)
		grep.Stdout = os.Stdout
		grep.Stderr = os.Stderr

		return grep.Run()
	},
}

func init() {
	rootCmd.AddCommand(grepCmd)
}
