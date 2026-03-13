package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dreikanter/notescli/note"
	"github.com/spf13/cobra"
)

var readCmd = &cobra.Command{
	Use:   "read <id|slug|filename>",
	Short: "Read a note by ID, slug, or filename",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root := mustNotesPath()
		notes, err := note.Scan(root)
		if err != nil {
			return err
		}

		n := note.Resolve(notes, args[0])
		if n == nil {
			return fmt.Errorf("note not found: %s", args[0])
		}

		data, err := os.ReadFile(filepath.Join(root, n.RelPath))
		if err != nil {
			return err
		}

		noFrontmatter, _ := cmd.Flags().GetBool("no-frontmatter")
		if noFrontmatter {
			data = note.StripFrontmatter(data)
		}

		_, err = os.Stdout.Write(data)
		return err
	},
}

func init() {
	readCmd.Flags().BoolP("no-frontmatter", "F", false, "exclude YAML frontmatter from output")
	rootCmd.AddCommand(readCmd)
}
