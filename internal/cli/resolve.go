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
  3. Absolute or relative file path
  4. Basename or slug — exact match on filename components

Alternatively, use filter flags (--type, --slug, --tag, --today) for
explicit attribute-based lookup. Flags cannot be combined with a
positional argument.`,
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root := mustNotesPath()

		today, _ := cmd.Flags().GetBool("today")
		types, _ := cmd.Flags().GetStringSlice("type")
		slug, _ := cmd.Flags().GetString("slug")
		tags, _ := cmd.Flags().GetStringSlice("tag")

		hasFilters := len(types) > 0 || slug != "" || len(tags) > 0

		if len(args) == 1 {
			if hasFilters {
				return fmt.Errorf("cannot combine positional argument with filter flags")
			}

			var date string
			if today {
				date = time.Now().Format("20060102")
			}

			n, err := note.ResolveRefDate(root, args[0], date)
			if err != nil {
				return err
			}

			fmt.Fprintln(cmd.OutOrStdout(), filepath.Join(root, n.RelPath))
			return nil
		}

		if !hasFilters && !today {
			return fmt.Errorf("specify a note by positional argument or filter flags (--type, --slug, --tag, --today)")
		}

		notes, err := note.Scan(root)
		if err != nil {
			return err
		}

		if today {
			notes = note.FilterByDate(notes, time.Now().Format("20060102"))
		}
		if len(types) > 0 {
			notes = note.FilterByTypes(notes, types)
		}
		if slug != "" {
			notes = note.FilterBySlug(notes, slug)
		}
		if len(tags) > 0 {
			notes, err = note.FilterByTags(notes, root, tags)
			if err != nil {
				return err
			}
		}

		if len(notes) == 0 {
			return fmt.Errorf("no notes found matching the given criteria")
		}

		fmt.Fprintln(cmd.OutOrStdout(), filepath.Join(root, notes[0].RelPath))
		return nil
	},
}

func registerResolveFlags() {
	resolveCmd.Flags().Bool("today", false, "only match notes created today")
	resolveCmd.Flags().StringSlice("type", nil, "filter by note type (repeatable)")
	resolveCmd.Flags().String("slug", "", "filter by slug")
	resolveCmd.Flags().StringSlice("tag", nil, "filter by tag (repeatable, all must match)")
}

func init() {
	registerResolveFlags()
	rootCmd.AddCommand(resolveCmd)
}
