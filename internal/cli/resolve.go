package cli

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/dreikanter/notes-cli/note"
	"github.com/spf13/cobra"
)

var resolveCmd = &cobra.Command{
	Use:   "resolve [<id|type|query>]",
	Short: "Resolve a note reference and print its absolute path",
	Long: `Resolve a note reference and print its absolute path.

With no arguments or flags, returns the most recent note.

With a positional argument, resolution follows this priority:
  1. Exact numeric ID (e.g. "8823") — all-digit queries match IDs only;
     an unknown numeric query errors instead of falling through
  2. Type with special behavior (todo, backlog, weekly) — most recent match
  3. Path (absolute or relative containing a separator) — exact match
  4. Slug substring — most recent note whose slug contains the query

Alternatively, use filter flags (--type, --slug, --tag, --today) for
explicit attribute-based lookup. --type, --slug, and --tag cannot be
combined with a positional argument; --today can, and restricts the
positional resolution to notes dated today.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := notesRoot()
		if err != nil {
			return err
		}
		f := readFilterFlags(cmd)

		if len(args) == 1 {
			if f.hasAttributeFilters() {
				return fmt.Errorf("cannot combine positional argument with filter flags")
			}

			var date string
			if f.Today {
				date = time.Now().Format(note.DateFormat)
			}

			n, err := note.ResolveRef(root, args[0], note.WithDate(date))
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), filepath.Join(root, n.RelPath))
			return nil
		}

		idx, err := note.Load(root, loadOptsFor(f))
		if err != nil {
			return err
		}

		entries := applyFilters(idx.Entries(), f)

		if len(entries) == 0 {
			if f.active() {
				return fmt.Errorf("no notes found matching filters: %s", f.describe())
			}
			return fmt.Errorf("no notes found")
		}

		fmt.Fprintln(cmd.OutOrStdout(), filepath.Join(root, entries[0].RelPath))
		return nil
	},
}

func registerResolveFlags() {
	addFilterFlags(resolveCmd)
}

func init() {
	registerResolveFlags()
	rootCmd.AddCommand(resolveCmd)
}
