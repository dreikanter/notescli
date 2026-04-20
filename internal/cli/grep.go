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

The following flags are injected automatically: -r (recursive), --include=*.md, --exclude-dir=.git. The notes path is appended as the last argument. Pass -i explicitly for case-insensitive search.`,
	DisableFlagParsing: true,
	SilenceErrors:      true,
	RunE: func(cmd *cobra.Command, args []string) error {
		for _, arg := range args {
			if arg == "--help" || arg == "-h" {
				return cmd.Help()
			}
		}

		root, err := notesRoot()
		if err != nil {
			return err
		}
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
