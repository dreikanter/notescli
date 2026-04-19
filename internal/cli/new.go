package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/dreikanter/notes-cli/note"
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
		upsert, _ := cmd.Flags().GetBool("upsert")

		if err := note.ValidateSlug(slug); err != nil {
			return err
		}

		if upsert && noteType == "" && slug == "" {
			return fmt.Errorf("--upsert requires --type or --slug")
		}

		root := mustNotesPath()

		// --upsert: check if today already has a matching note
		if upsert {
			today := time.Now().Format("20060102")
			notes, err := note.Scan(root)
			if err != nil {
				return err
			}
			notes = note.FilterByDate(notes, today)
			if noteType != "" {
				notes = note.FilterByTypes(notes, []string{noteType})
			}
			if slug != "" {
				notes = note.FilterBySlug(notes, slug)
			}
			if len(notes) > 0 {
				fmt.Fprintln(cmd.OutOrStdout(), filepath.Join(root, notes[0].RelPath))
				return nil
			}
		}

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
	newCmd.Flags().String("type", "", "note type (free-form; todo/backlog/weekly get special behavior)")
	newCmd.Flags().StringSlice("tag", nil, "tag for frontmatter (repeatable)")
	newCmd.Flags().String("description", "", "description for frontmatter")
	newCmd.Flags().String("title", "", "title for frontmatter")
	newCmd.Flags().Bool("public", false, "mark note as public in frontmatter")
	newCmd.Flags().Bool("private", false, "mark note as private in frontmatter (default; overrides --public)")
	newCmd.Flags().Bool("upsert", false, "return existing note if today already has one matching --type/--slug")
	rootCmd.AddCommand(newCmd)
}
