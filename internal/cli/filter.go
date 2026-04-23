package cli

import (
	"fmt"
	"strings"
	"time"

	"github.com/dreikanter/notes-cli/note"
	"github.com/spf13/cobra"
)

// stderrLogger returns a note.Logger that writes non-fatal warnings from
// Load/Scan/Reload to cmd's stderr. The note package itself no longer writes
// to os.Stderr — CLI commands wire this at the edge.
func stderrLogger(cmd *cobra.Command) note.Logger {
	return func(err error) {
		fmt.Fprintf(cmd.ErrOrStderr(), "warn: %v\n", err)
	}
}

// resolveRef loads the index with WithFrontmatter(false) and resolves query
// through Index.Resolve. Misses wrap note.ErrNotFound so callers can match
// with errors.Is. Commands that already hold an Index should call
// idx.Resolve directly.
func resolveRef(cmd *cobra.Command, root, query string, opts ...note.ResolveOption) (note.Ref, error) {
	idx, err := note.Load(root, note.WithFrontmatter(false), note.WithLogger(stderrLogger(cmd)))
	if err != nil {
		return note.Ref{}, err
	}
	e, ok, err := idx.Resolve(query, opts...)
	if err != nil {
		return note.Ref{}, err
	}
	if !ok {
		return note.Ref{}, fmt.Errorf("%w: %s", note.ErrNotFound, strings.TrimSpace(query))
	}
	return e.Ref, nil
}

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
// touches filename-derived fields, so the frontmatter read is skipped. The
// stderr logger is attached so the note package's per-note and subdirectory
// warnings surface to the user.
func loadOptsFor(cmd *cobra.Command, f filterOpts) []note.LoadOption {
	return []note.LoadOption{
		note.WithFrontmatter(len(f.Tags) > 0),
		note.WithLogger(stderrLogger(cmd)),
	}
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
