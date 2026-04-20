package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dreikanter/notes-cli/note"
	"github.com/spf13/cobra"
)

var readCmd = &cobra.Command{
	Use:   "read [<id|type|query>]",
	Short: "Read a note by ref or filter flags",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := notesRoot()
		if err != nil {
			return err
		}
		f := readFilterFlags(cmd)
		noFrontmatter, _ := cmd.Flags().GetBool("no-frontmatter")

		var relPath string

		if len(args) == 1 {
			if f.active() {
				return fmt.Errorf("cannot combine positional argument with filter flags")
			}
			n, err := note.ResolveRef(root, args[0])
			if err != nil {
				return err
			}
			relPath = n.RelPath
		} else if f.active() {
			notes, err := note.Scan(root)
			if err != nil {
				return err
			}

			notes, err = applyFilters(notes, root, f)
			if err != nil {
				return err
			}

			if len(notes) == 0 {
				return fmt.Errorf("no notes found matching filters: %s", f.describe())
			}
			relPath = notes[0].RelPath
		} else {
			return fmt.Errorf("specify a note by positional argument or filter flags (--type, --slug, --tag, --today)")
		}

		data, err := os.ReadFile(filepath.Join(root, relPath))
		if err != nil {
			return err
		}

		if noFrontmatter {
			data = note.StripFrontmatter(data)
		}

		_, err = cmd.OutOrStdout().Write(data)
		return err
	},
}

func registerReadFlags() {
	addFilterFlags(readCmd)
	readCmd.Flags().Bool("no-frontmatter", false, "exclude YAML frontmatter from output")
}

func init() {
	registerReadFlags()
	rootCmd.AddCommand(readCmd)
}
