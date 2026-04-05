package cli

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	"github.com/dreikanter/notescli/note"
	"github.com/spf13/cobra"
)

var editCmd = &cobra.Command{
	Use:   "edit <id|path|basename|slug|type>",
	Short: "Open a note in your editor",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root := mustNotesPath()
		n, err := note.ResolveRef(root, args[0])
		if err != nil {
			return err
		}

		editor := os.Getenv("VISUAL")
		if editor == "" {
			editor = os.Getenv("EDITOR")
		}
		if editor == "" {
			return fmt.Errorf("no editor configured: set $EDITOR or $VISUAL")
		}

		path := filepath.Join(root, n.RelPath)
		ec := exec.Command(editor, path)
		ec.Stdin = os.Stdin
		ec.Stdout = os.Stdout
		ec.Stderr = os.Stderr

		return ec.Run()
	},
}

func init() {
	rootCmd.AddCommand(editCmd)
}
