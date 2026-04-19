package cli

import (
	"time"

	"github.com/dreikanter/notes-cli/note"
	"github.com/spf13/cobra"
)

// filterOpts holds the common filter flag values.
type filterOpts struct {
	Today bool
	Types []string
	Slug  string
	Tags  []string
}

// readFilterFlags reads the common filter flags from a cobra command.
func readFilterFlags(cmd *cobra.Command) filterOpts {
	today, _ := cmd.Flags().GetBool("today")
	types, _ := cmd.Flags().GetStringSlice("type")
	slug, _ := cmd.Flags().GetString("slug")
	tags, _ := cmd.Flags().GetStringSlice("tag")
	return filterOpts{Today: today, Types: types, Slug: slug, Tags: tags}
}

func (f filterOpts) active() bool {
	return f.Today || f.hasAttributeFilters()
}

func (f filterOpts) hasAttributeFilters() bool {
	return len(f.Types) > 0 || f.Slug != "" || len(f.Tags) > 0
}

// applyFilters applies the common filter pipeline to a list of notes.
func applyFilters(notes []note.Note, root string, f filterOpts) ([]note.Note, error) {
	if f.Today {
		notes = note.FilterByDate(notes, time.Now().Format("20060102"))
	}
	if len(f.Types) > 0 {
		notes = note.FilterByTypes(notes, f.Types)
	}
	if f.Slug != "" {
		notes = note.FilterBySlug(notes, f.Slug)
	}
	if len(f.Tags) > 0 {
		var err error
		notes, err = note.FilterByTags(notes, root, f.Tags)
		if err != nil {
			return nil, err
		}
	}
	return notes, nil
}

// addFilterFlags registers the common filter flags on a command.
func addFilterFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("today", false, "only match notes created today")
	cmd.Flags().StringSlice("type", nil, "filter by note type from filename suffix (repeatable; use update --sync-filename to reconcile after fm edits)")
	cmd.Flags().String("slug", "", "filter by slug")
	cmd.Flags().StringSlice("tag", nil, "filter by tag (repeatable, all must match)")
}
