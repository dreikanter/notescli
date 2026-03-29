package cli

import (
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/dreikanter/notescli/note"
	"github.com/spf13/cobra"
)

var newCmd = &cobra.Command{
	Use:   "new",
	Short: "Create a new note",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		slug, _ := cmd.Flags().GetString("slug")
		noteType, _ := cmd.Flags().GetString("type")
		tags, _ := cmd.Flags().GetStringSlice("tag")
		description, _ := cmd.Flags().GetString("description")
		title, _ := cmd.Flags().GetString("title")
		publicFlag, _ := cmd.Flags().GetBool("public")
		privateFlag, _ := cmd.Flags().GetBool("private")

		if noteType != "" && !note.IsKnownType(noteType) {
			return fmt.Errorf("unknown note type %q (valid types: %s)", noteType, strings.Join(note.KnownTypes, ", "))
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

		public := publicFlag && !privateFlag
		fullPath, err := createNote(createNoteParams{
			Root:        root,
			Slug:        slug,
			Type:        noteType,
			Tags:        tags,
			Title:       title,
			Description: description,
			Public:      public,
			Body:        body,
		})
		if err != nil {
			return err
		}

		fmt.Fprintln(cmd.OutOrStdout(), fullPath)
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
	newCmd.Flags().String("slug", "", "descriptive slug appended to filename")
	newCmd.Flags().String("type", "", "note type (todo, backlog, weekly)")
	newCmd.Flags().StringSlice("tag", nil, "tag for frontmatter (repeatable)")
	newCmd.Flags().String("description", "", "description for frontmatter")
	newCmd.Flags().String("title", "", "title for frontmatter")
	newCmd.Flags().Bool("public", false, "mark note as public in frontmatter")
	newCmd.Flags().Bool("private", false, "mark note as private in frontmatter (default; overrides --public)")
	rootCmd.AddCommand(newCmd)
}
