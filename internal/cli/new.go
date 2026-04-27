package cli

import (
	"errors"
	"fmt"
	"time"

	"github.com/dreikanter/notesctl/note"
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

		store, err := notesStore()
		if err != nil {
			return err
		}

		if upsert {
			if existing, found, err := findUpsertEntry(store, noteType, slug); err != nil {
				return err
			} else if found {
				fmt.Fprintln(cmd.OutOrStdout(), store.AbsPath(existing))
				return nil
			}
		}

		body, err := readStdinBody(cmd)
		if err != nil {
			return err
		}

		entry := note.Entry{
			Meta: note.Meta{
				Title:       title,
				Slug:        slug,
				Type:        noteType,
				Tags:        tags,
				Description: description,
				Public:      publicFlag,
			},
			Body: body,
		}
		saved, err := store.Put(entry)
		if err != nil {
			return err
		}

		fmt.Fprintln(cmd.OutOrStdout(), store.AbsPath(saved))
		return nil
	},
}

// findUpsertEntry looks for today's note matching noteType and slug.
// Returns (entry, true, nil) on hit, (zero, false, nil) on clean miss, and
// a non-nil error only for I/O failures.
func findUpsertEntry(store note.Store, noteType, slug string) (note.Entry, bool, error) {
	opts := []note.QueryOpt{note.WithExactDate(time.Now())}
	if noteType != "" {
		opts = append(opts, note.WithType(noteType))
	}
	if slug != "" {
		opts = append(opts, note.WithSlug(slug))
	}
	entry, err := store.Find(opts...)
	if err != nil {
		if errors.Is(err, note.ErrNotFound) {
			return note.Entry{}, false, nil
		}
		return note.Entry{}, false, err
	}
	return entry, true, nil
}

func registerNewFlags() {
	newCmd.Flags().String("slug", "", "descriptive slug for the note")
	newCmd.Flags().String("type", "", "note type (free-form; todo/backlog/weekly get special handling)")
	newCmd.Flags().StringSlice("tag", nil, "tag (repeatable)")
	newCmd.Flags().String("description", "", "note description")
	newCmd.Flags().String("title", "", "note title")
	newCmd.Flags().Bool("public", false, "mark note as public (private is the default)")
	newCmd.Flags().Bool("upsert", false, "reuse today's note if one already matches --type/--slug")
}

func init() {
	registerNewFlags()
	rootCmd.AddCommand(newCmd)
}
