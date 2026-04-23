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
		upsert, _ := cmd.Flags().GetBool("upsert")

		if err := note.ValidateSlug(slug); err != nil {
			return err
		}

		if upsert && noteType == "" && slug == "" {
			return fmt.Errorf("--upsert requires --type or --slug")
		}

		root, err := notesRoot()
		if err != nil {
			return err
		}

		// --upsert: check if today already has a matching note
		if upsert {
			today := time.Now().Format(note.DateFormat)
			idx, err := note.Load(root, note.WithFrontmatter(false))
			if err != nil {
				return err
			}
			entries := note.FilterByDate(idx.Entries(), today)
			if noteType != "" {
				entries = note.FilterByTypes(entries, []string{noteType})
			}
			if slug != "" {
				entries = note.FilterBySlug(entries, slug)
			}
			if len(entries) > 0 {
				fmt.Fprintln(cmd.OutOrStdout(), filepath.Join(root, entries[0].RelPath))
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

		fullPath, err := createNote(createNoteParams{
			Root:        root,
			Slug:        slug,
			Type:        noteType,
			Tags:        tags,
			Title:       title,
			Description: description,
			Public:      publicFlag,
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

func registerNewFlags() {
	newCmd.Flags().String("slug", "", "descriptive slug appended to filename")
	newCmd.Flags().String("type", "", "note type (free-form; todo/backlog/weekly get special behavior)")
	newCmd.Flags().StringSlice("tag", nil, "tag for frontmatter (repeatable)")
	newCmd.Flags().String("description", "", "description for frontmatter")
	newCmd.Flags().String("title", "", "title for frontmatter")
	newCmd.Flags().Bool("public", false, "mark note as public in frontmatter")
	newCmd.Flags().Bool("private", false, "mark note as private in frontmatter (default)")
	newCmd.Flags().Bool("upsert", false, "return existing note if today already has one matching --type/--slug")
	newCmd.MarkFlagsMutuallyExclusive("public", "private")
}

func init() {
	registerNewFlags()
	rootCmd.AddCommand(newCmd)
}
