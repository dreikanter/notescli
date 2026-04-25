package cli

import (
	"fmt"
	"time"

	"github.com/dreikanter/notesctl/note"
	"github.com/spf13/cobra"
)

var lsCmd = &cobra.Command{
	Use:   "ls",
	Short: "List note IDs, newest first",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		lsLimit, _ := cmd.Flags().GetInt("limit")
		noteType, _ := cmd.Flags().GetString("type")
		slug, _ := cmd.Flags().GetString("slug")
		tags, _ := cmd.Flags().GetStringSlice("tag")
		today, _ := cmd.Flags().GetBool("today")

		store, err := notesStore()
		if err != nil {
			return err
		}

		ids, err := lsIDs(store, noteType, slug, tags, today)
		if err != nil {
			return err
		}

		if lsLimit > 0 && len(ids) > lsLimit {
			ids = ids[:lsLimit]
		}
		out := cmd.OutOrStdout()
		for _, id := range ids {
			fmt.Fprintln(out, id)
		}
		return nil
	},
}

// lsIDs returns the IDs to print. With no filter flags it takes the fast
// directory-scan path via Store.IDs; otherwise it builds a QueryOpt list
// and delegates to Store.All.
func lsIDs(store note.Store, noteType, slug string, tags []string, today bool) ([]int, error) {
	if noteType == "" && slug == "" && len(tags) == 0 && !today {
		return store.IDs()
	}
	var opts []note.QueryOpt
	if noteType != "" {
		opts = append(opts, note.WithType(noteType))
	}
	if slug != "" {
		opts = append(opts, note.WithSlug(slug))
	}
	for _, t := range tags {
		opts = append(opts, note.WithTag(t))
	}
	if today {
		opts = append(opts, note.WithExactDate(time.Now()))
	}
	entries, err := store.All(opts...)
	if err != nil {
		return nil, err
	}
	ids := make([]int, len(entries))
	for i, e := range entries {
		ids[i] = e.ID
	}
	return ids, nil
}

func registerLsFlags() {
	lsCmd.Flags().Int("limit", 0, "maximum number of IDs to list (0 = no limit)")
	lsCmd.Flags().String("type", "", "filter by note type")
	lsCmd.Flags().String("slug", "", "filter by exact slug")
	lsCmd.Flags().StringSlice("tag", nil, "filter by tag (repeatable, all must match)")
	lsCmd.Flags().Bool("today", false, "only list notesctl created today")
}

func init() {
	registerLsFlags()
	rootCmd.AddCommand(lsCmd)
}
