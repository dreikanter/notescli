package cli

import (
	"errors"
	"fmt"
	"strconv"
	"time"

	"github.com/dreikanter/notesctl/note"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var updateCmd = &cobra.Command{
	Use:   "update <id>",
	Short: "Update note frontmatter (file is renamed automatically when slug, type, or date changes)",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("id must be an integer: %s", args[0])
		}

		tags, _ := cmd.Flags().GetStringSlice("tag")
		noTags, _ := cmd.Flags().GetBool("no-tags")
		title, _ := cmd.Flags().GetString("title")
		description, _ := cmd.Flags().GetString("description")
		slug, _ := cmd.Flags().GetString("slug")
		noSlug, _ := cmd.Flags().GetBool("no-slug")
		noteType, _ := cmd.Flags().GetString("type")
		noType, _ := cmd.Flags().GetBool("no-type")
		dateStr, _ := cmd.Flags().GetString("date")

		hasFlag := false
		cmd.LocalFlags().VisitAll(func(f *pflag.Flag) {
			if f.Changed {
				hasFlag = true
			}
		})
		if !hasFlag {
			return fmt.Errorf("at least one update flag is required")
		}

		if cmd.Flags().Changed("slug") {
			if err := note.ValidateSlug(slug); err != nil {
				return err
			}
		}
		var newDate time.Time
		if cmd.Flags().Changed("date") {
			parsed, err := time.Parse(note.DateFormat, dateStr)
			if err != nil {
				return fmt.Errorf("invalid date %q: expected %s", dateStr, note.DateFormat)
			}
			newDate = parsed
		}

		store, err := notesStore()
		if err != nil {
			return err
		}
		entry, err := store.Get(id)
		if err != nil {
			if errors.Is(err, note.ErrNotFound) {
				return fmt.Errorf("note %d not found", id)
			}
			return err
		}

		if cmd.Flags().Changed("title") {
			entry.Meta.Title = title
		}
		if cmd.Flags().Changed("description") {
			entry.Meta.Description = description
		}
		if noTags {
			entry.Meta.Tags = nil
		} else if cmd.Flags().Changed("tag") {
			entry.Meta.Tags = tags
		}
		if noSlug {
			entry.Meta.Slug = ""
		} else if cmd.Flags().Changed("slug") {
			entry.Meta.Slug = slug
		}
		if noType {
			entry.Meta.Type = ""
		} else if cmd.Flags().Changed("type") {
			entry.Meta.Type = noteType
		}
		if cmd.Flags().Changed("private") {
			v, _ := cmd.Flags().GetBool("private")
			entry.Meta.Public = !v
		} else if cmd.Flags().Changed("public") {
			v, _ := cmd.Flags().GetBool("public")
			entry.Meta.Public = v
		}
		if cmd.Flags().Changed("date") {
			entry.Meta.CreatedAt = newDate
		}

		saved, err := store.Put(entry)
		if err != nil {
			return err
		}

		fmt.Fprintln(cmd.OutOrStdout(), store.AbsPath(saved))
		return nil
	},
}

func registerUpdateFlags() {
	updateCmd.Flags().StringSlice("tag", nil, "tag (repeatable); replaces existing tags")
	updateCmd.Flags().Bool("no-tags", false, "remove all tags")
	updateCmd.Flags().String("title", "", "note title (empty string clears it)")
	updateCmd.Flags().String("description", "", "note description (empty string clears it)")
	updateCmd.Flags().String("slug", "", "set slug; file is renamed to match")
	updateCmd.Flags().Bool("no-slug", false, "remove slug; file is renamed to match")
	updateCmd.Flags().String("type", "", "set note type; file is renamed to match")
	updateCmd.Flags().Bool("no-type", false, "remove type; file is renamed to match")
	updateCmd.Flags().Bool("public", false, "mark note as public")
	updateCmd.Flags().Bool("private", false, "mark note as private")
	updateCmd.Flags().String("date", "", "move the note to this date (YYYYMMDD); file is moved to match")
	updateCmd.MarkFlagsMutuallyExclusive("slug", "no-slug")
	updateCmd.MarkFlagsMutuallyExclusive("type", "no-type")
	updateCmd.MarkFlagsMutuallyExclusive("tag", "no-tags")
	updateCmd.MarkFlagsMutuallyExclusive("public", "private")
}

func init() {
	registerUpdateFlags()
	rootCmd.AddCommand(updateCmd)
}
