package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

var rgCmd = &cobra.Command{
	Use:                "rg [flags] <pattern>",
	Short:              "Search note contents using ripgrep",
	Long:               "Search note contents using ripgrep (rg). Only .md files are searched.",
	DisableFlagParsing: true,
	SilenceErrors:      true,
	RunE: func(cmd *cobra.Command, args []string) error {
		if _, err := exec.LookPath("rg"); err != nil {
			return fmt.Errorf("ripgrep (rg) is not installed; install it from https://github.com/BurntSushi/ripgrep")
		}

		root := mustNotesPath()
		rgArgs := append([]string{"--glob", "*.md"}, args...)
		rgArgs = append(rgArgs, root)

		rg := exec.Command("rg", rgArgs...)
		rg.Stdout = os.Stdout
		rg.Stderr = os.Stderr

		return rg.Run()
	},
}

func init() {
	rootCmd.AddCommand(rgCmd)
}
