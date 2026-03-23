package cli

import (
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var grepCmd = &cobra.Command{
	Use:                "grep [flags] <pattern>",
	Short:              "Search note contents using grep",
	Long:               "Search note contents using grep. Only .md files are searched; .git directories are excluded.",
	DisableFlagParsing: true,
	SilenceErrors:      true,
	RunE: func(cmd *cobra.Command, args []string) error {
		root := mustNotesPath()
		grepArgs := append([]string{"-r", "--include=*.md", "--exclude-dir=.git"}, args...)
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
