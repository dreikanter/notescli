package cli

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

// runExternalSearch is the shared dispatch skeleton for grep and rg.
// It handles help detection, optional PATH check, root resolution, and exec.
// buildArgs constructs the full argument list from the notes root and the
// pass-through args. When notInstalled is non-empty, a PATH check is performed
// first; a missing binary returns an error with that message.
func runExternalSearch(
	cmd *cobra.Command,
	args []string,
	tool, notInstalled string,
	buildArgs func(root string, args []string) []string,
) error {
	for _, arg := range args {
		if arg == "--help" || arg == "-h" {
			return cmd.Help()
		}
	}
	if notInstalled != "" {
		if _, err := exec.LookPath(tool); err != nil {
			return fmt.Errorf("%s", notInstalled)
		}
	}
	root, err := notesRoot()
	if err != nil {
		return err
	}
	c := exec.Command(tool, buildArgs(root, args)...)
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
	return c.Run()
}
