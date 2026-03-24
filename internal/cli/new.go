package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/dreikanter/notescli/note"
	"github.com/spf13/cobra"
)

var (
	newSlug        string
	newType        string
	newTags        []string
	newDescription string
	newTitle       string
)

var newCmd = &cobra.Command{
	Use:   "new",
	Short: "Create a new note",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		if newType != "" && !note.IsKnownType(newType) {
			return fmt.Errorf("unknown note type %q (valid types: %s)", newType, strings.Join(note.KnownTypes, ", "))
		}

		root := mustNotesPath()

		var body string
		if !isTerminal(os.Stdin) {
			data, err := io.ReadAll(os.Stdin)
			if err != nil {
				return fmt.Errorf("cannot read stdin: %w", err)
			}
			body = string(data)
		}

		fullPath, err := createNote(createNoteParams{
			Root:        root,
			Slug:        newSlug,
			Type:        newType,
			Tags:        newTags,
			Title:       newTitle,
			Description: newDescription,
			Body:        body,
		})
		if err != nil {
			return err
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
	newCmd.Flags().StringVar(&newSlug, "slug", "", "descriptive slug appended to filename")
	newCmd.Flags().StringVar(&newType, "type", "", "note type (todo, backlog, weekly)")
	newCmd.Flags().StringArrayVar(&newTags, "tag", nil, "tag for frontmatter (repeatable)")
	newCmd.Flags().StringVar(&newDescription, "description", "", "description for frontmatter")
	newCmd.Flags().StringVar(&newTitle, "title", "", "title for frontmatter")
	rootCmd.AddCommand(newCmd)
}
