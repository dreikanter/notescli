package cli

import (
	"errors"
	"fmt"
	"strconv"

	"github.com/dreikanter/notesctl/note"
	"github.com/spf13/cobra"
)

var resolveCmd = &cobra.Command{
	Use:   "resolve",
	Short: "Print the absolute path of a note by explicit lookup flag",
	Long: `Resolve a note by an explicit lookup flag and print its absolute path.

  notes resolve               - most recent note
  notes resolve --id <id>     - exact ID
  notes resolve --type <t>    - most recent note of that type
  notes resolve --slug <s>    - most recent note with that slug
  notes resolve --tag <t>     - most recent note with that tag

Exactly one lookup flag (or none) may be provided.`,
	Args: cobra.NoArgs,
	RunE: resolveRunE,
}

func resolveRunE(cmd *cobra.Command, _ []string) error {
	idStr, _ := cmd.Flags().GetString("id")
	noteType, _ := cmd.Flags().GetString("type")
	slug, _ := cmd.Flags().GetString("slug")
	tag, _ := cmd.Flags().GetString("tag")

	if err := ensureSingleLookupFlag(idStr, noteType, slug, tag); err != nil {
		return err
	}

	store, err := notesStore()
	if err != nil {
		return err
	}

	entry, err := lookupEntry(store, idStr, noteType, slug, tag)
	if err != nil {
		if errors.Is(err, note.ErrNotFound) {
			return fmt.Errorf("no matching note found")
		}
		return err
	}

	fmt.Fprintln(cmd.OutOrStdout(), store.AbsPath(entry))
	return nil
}

// ensureSingleLookupFlag reports an error when more than one of id/type/slug/tag
// is set; zero or one is allowed.
func ensureSingleLookupFlag(idStr, noteType, slug, tag string) error {
	count := 0
	for _, v := range []string{idStr, noteType, slug, tag} {
		if v != "" {
			count++
		}
	}
	if count > 1 {
		return fmt.Errorf("resolve accepts at most one of --id, --type, --slug, --tag")
	}
	return nil
}

// lookupEntry dispatches to the correct Store call for the set flag. An
// empty flag set returns the newest note.
func lookupEntry(store *note.OSStore, idStr, noteType, slug, tag string) (note.Entry, error) {
	switch {
	case idStr != "":
		id, err := strconv.Atoi(idStr)
		if err != nil {
			return note.Entry{}, fmt.Errorf("--id must be an integer: %s", idStr)
		}
		return store.Get(id)
	case noteType != "":
		return store.Find(note.WithType(noteType))
	case slug != "":
		return store.Find(note.WithSlug(slug))
	case tag != "":
		return store.Find(note.WithTag(tag))
	default:
		return store.Find()
	}
}

func registerResolveFlags() {
	resolveCmd.Flags().String("id", "", "resolve by exact numeric ID")
	resolveCmd.Flags().String("type", "", "resolve the most recent note with the given type")
	resolveCmd.Flags().String("slug", "", "resolve the most recent note with the given slug")
	resolveCmd.Flags().String("tag", "", "resolve the most recent note with the given tag")
}

func init() {
	registerResolveFlags()
	rootCmd.AddCommand(resolveCmd)
}
