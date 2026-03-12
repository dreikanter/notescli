package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/dreikanter/notescli/note"
	"github.com/spf13/cobra"
)

var (
	newSlug        string
	newTags        []string
	newDescription string
)

var newCmd = &cobra.Command{
	Use:   "new",
	Short: "Create a new note",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		root := mustNotesPath()
		today := time.Now().Format("20060102")

		id, err := note.NextID(root)
		if err != nil {
			return err
		}

		filename := note.NoteFilename(today, id, newSlug)
		dir := note.NoteDirPath(root, today)

		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("cannot create directory %s: %w", dir, err)
		}

		fullPath := filepath.Join(dir, filename)

		var content string

		// Build frontmatter if tags or description provided
		fm := note.BuildFrontmatter("", newTags, newDescription)
		content = fm

		// Read from stdin if piped
		if !isTerminal(os.Stdin) {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("cannot read stdin: %w", err)
			}
			content += string(data)
		}

		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("cannot write note: %w", err)
		}

		fmt.Println(fullPath)
		return nil
	},
}

func isTerminal(f *os.File) bool {
	fi, err := f.Stat()
	if err != nil {
		return true
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

func init() {
	newCmd.Flags().StringVar(&newSlug, "slug", "", "slug appended to filename")
	newCmd.Flags().StringArrayVar(&newTags, "tag", nil, "tag for frontmatter (repeatable)")
	newCmd.Flags().StringVar(&newDescription, "description", "", "description for frontmatter")
	rootCmd.AddCommand(newCmd)
}
