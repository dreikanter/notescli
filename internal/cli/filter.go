package cli

import (
	"fmt"
	"strings"
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

// describe returns a human-readable summary of the active filters, e.g.
// "type=[todo] today=true". Returns an empty string if no filters are set.
func (f filterOpts) describe() string {
	var parts []string
	if len(f.Types) > 0 {
		parts = append(parts, fmt.Sprintf("type=[%s]", strings.Join(f.Types, ",")))
	}
	if f.Slug != "" {
		parts = append(parts, fmt.Sprintf("slug=%s", f.Slug))
	}
	if len(f.Tags) > 0 {
		parts = append(parts, fmt.Sprintf("tag=[%s]", strings.Join(f.Tags, ",")))
	}
	if f.Today {
		parts = append(parts, "today=true")
	}
	return strings.Join(parts, " ")
}

// loadOptsFor picks Load options matching the fields this filter set touches.
// Tag filters need merged frontmatter+body tags; every other filter only
// touches filename-derived fields, so the frontmatter read is skipped.
func loadOptsFor(f filterOpts) note.LoadOption {
	return note.WithFrontmatter(len(f.Tags) > 0)
}

// applyFilters applies the common filter pipeline to a list of entries.
// All stages operate on []Entry, so the tag filter reads tags directly from
// the entries (populated once during Load) rather than re-scanning the store.
func applyFilters(entries []note.Entry, f filterOpts) []note.Entry {
	if f.Today {
		entries = note.FilterByDate(entries, time.Now().Format(note.DateFormat))
	}
	if len(f.Types) > 0 {
		entries = note.FilterByTypes(entries, f.Types)
	}
	if f.Slug != "" {
		entries = note.FilterBySlug(entries, f.Slug)
	}
	if len(f.Tags) > 0 {
		entries = note.FilterByTags(entries, f.Tags)
	}
	return entries
}

// addFilterFlags registers the common filter flags on a command.
func addFilterFlags(cmd *cobra.Command) {
	cmd.Flags().Bool("today", false, "only match notes created today")
	cmd.Flags().StringSlice("type", nil, "filter by note type from filename suffix (repeatable; use update --sync-filename to reconcile after fm edits)")
	cmd.Flags().String("slug", "", "filter by slug")
	cmd.Flags().StringSlice("tag", nil, "filter by tag (repeatable, all must match)")
}
