package cli

import (
	"fmt"
	"path/filepath"
	"time"

	"github.com/dreikanter/notescli/note"
	"github.com/spf13/cobra"
)

var resolveCmd = &cobra.Command{
	Use:   "resolve [<id|type|query>]",
	Short: "Resolve a note reference and print its absolute path",
	Long: `Resolve a note reference and print its absolute path.

With a positional argument, resolution follows this priority:
  1. Exact numeric ID (e.g. "8823")
  2. Exact note type (todo, backlog, weekly) — most recent match
  3. Path substring — most recent note whose path contains the query
     (covers slugs, basenames, date fragments, relative paths, etc.)

Alternatively, use filter flags (--type, --slug, --tag, --today) for
explicit attribute-based lookup. Flags cannot be combined with a
positional argument.`,
	Args: cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root := mustNotesPath()
		f := readFilterFlags(cmd)

		hasNonTodayFilters := len(f.Types) > 0 || f.Slug != "" || len(f.Tags) > 0

		if len(args) == 1 {
			if hasNonTodayFilters {
				return fmt.Errorf("cannot combine positional argument with filter flags")
			}

			var date string
			if f.Today {
				date = time.Now().Format("20060102")
			}

			n, err := note.ResolveRefDate(root, args[0], date)
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), filepath.Join(root, n.RelPath))
			return nil
		}

		if !f.active() {
			return fmt.Errorf("specify a note by positional argument or filter flags (--type, --slug, --tag, --today)")
		}

		notes, err := note.Scan(root)
		if err != nil {
			return err
		}

		notes, err = applyFilters(notes, root, f)
		if err != nil {
			return err
		}

		if len(notes) == 0 {
			return fmt.Errorf("no notes found matching the given criteria")
		}

		fmt.Fprintln(cmd.OutOrStdout(), filepath.Join(root, notes[0].RelPath))
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
