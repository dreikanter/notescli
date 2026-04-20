package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/dreikanter/notes-cli/note"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var updateCmd = &cobra.Command{
	Use:   "update <id|type|query>",
	Short: "Update frontmatter; use --sync-filename to reconcile the filename",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		updateTags, _ := cmd.Flags().GetStringSlice("tag")
		updateNoTags, _ := cmd.Flags().GetBool("no-tags")
		updateTitle, _ := cmd.Flags().GetString("title")
		updateDescription, _ := cmd.Flags().GetString("description")
		updateSlug, _ := cmd.Flags().GetString("slug")
		updateNoSlug, _ := cmd.Flags().GetBool("no-slug")
		updateType, _ := cmd.Flags().GetString("type")
		updateNoType, _ := cmd.Flags().GetBool("no-type")
		updatePrivate, _ := cmd.Flags().GetBool("private")
		syncFilename, _ := cmd.Flags().GetBool("sync-filename")

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
			if err := note.ValidateSlug(updateSlug); err != nil {
				return err
			}
		}

		root, err := notesRoot()
		if err != nil {
			return err
		}
		n, err := note.ResolveRef(root, args[0])
		if err != nil {
			return err
		}

		oldPath := filepath.Join(root, n.RelPath)
		data, err := os.ReadFile(oldPath)
		if err != nil {
			return fmt.Errorf("cannot read note: %w", err)
		}

		updated, body, err := note.ParseNote(data)
		if err != nil {
			return fmt.Errorf("%s: %w", oldPath, err)
		}

		if cmd.Flags().Changed("title") {
			updated.Title = updateTitle
		}
		if cmd.Flags().Changed("description") {
			updated.Description = updateDescription
		}
		if updateNoTags {
			updated.Tags = nil
		} else if cmd.Flags().Changed("tag") {
			updated.Tags = updateTags
		}

		if updateNoSlug {
			updated.Slug = ""
		} else if cmd.Flags().Changed("slug") {
			updated.Slug = updateSlug
		}
		if updatePrivate {
			updated.Public = false
		} else if cmd.Flags().Changed("public") {
			updated.Public = true
		}
		if updateNoType {
			updated.Type = ""
		} else if cmd.Flags().Changed("type") {
			updated.Type = updateType
		}

		// Any non-sync flag => rewrite the frontmatter in place (no rename).
		contentChanged := cmd.Flags().Changed("title") ||
			cmd.Flags().Changed("description") ||
			cmd.Flags().Changed("tag") || updateNoTags ||
			cmd.Flags().Changed("slug") || updateNoSlug ||
			cmd.Flags().Changed("type") || updateNoType ||
			cmd.Flags().Changed("public") || updatePrivate

		if contentChanged {
			newContent := note.FormatNote(updated, body)
			if err := writeAtomic(oldPath, newContent); err != nil {
				return err
			}
		}

		// --sync-filename: reconcile filename to match (already-updated) frontmatter.
		// When frontmatter is silent on slug/type AND the user didn't touch
		// those flags, fall back to the filename-reported value so the rename
		// is a no-op instead of stripping a still-valid cache suffix.
		newPath := oldPath
		if syncFilename {
			id, err := strconv.Atoi(n.ID)
			if err != nil {
				return fmt.Errorf("invalid note id %q: %w", n.ID, err)
			}
			syncSlug := updated.Slug
			if syncSlug == "" && !cmd.Flags().Changed("slug") && !updateNoSlug {
				syncSlug = n.Slug
			}
			syncType := updated.Type
			if syncType == "" && !cmd.Flags().Changed("type") && !updateNoType {
				syncType = n.Type
			}
			newFilename := note.NoteFilename(n.Date, id, syncSlug, syncType)
			dir := filepath.Dir(oldPath)
			newPath = filepath.Join(dir, newFilename)
			if newPath != oldPath {
				// os.Link atomically reserves the target: returns EEXIST if it
				// already exists, which os.Rename on Unix would silently clobber.
				if err := os.Link(oldPath, newPath); err != nil {
					if errors.Is(err, os.ErrExist) {
						return fmt.Errorf("target note already exists: %s", newPath)
					}
					return fmt.Errorf("cannot link note: %w", err)
				}
				if err := os.Remove(oldPath); err != nil {
					// Roll back the link so we don't leave both paths pointing
					// to the same inode. Best-effort: a cleanup failure here
					// isn't surfaced because the original error is more useful.
					_ = os.Remove(newPath)
					return fmt.Errorf("cannot remove old note: %w", err)
				}
			}
		}

		fmt.Fprintln(cmd.OutOrStdout(), newPath)
		return nil
	},
}

func init() {
	updateCmd.Flags().StringSlice("tag", nil, "tag for frontmatter (repeatable); replaces existing tags")
	updateCmd.Flags().Bool("no-tags", false, "remove all tags from frontmatter")
	updateCmd.Flags().String("title", "", "title for frontmatter (empty string clears it)")
	updateCmd.Flags().String("description", "", "description for frontmatter (empty string clears it)")
	updateCmd.Flags().String("slug", "", "update slug in frontmatter; does not rename the file")
	updateCmd.Flags().Bool("no-slug", false, "remove slug from frontmatter")
	updateCmd.Flags().String("type", "", "update type in frontmatter; does not rename the file")
	updateCmd.Flags().Bool("no-type", false, "remove type from frontmatter")
	updateCmd.Flags().Bool("public", false, "mark note as public in frontmatter")
	updateCmd.Flags().Bool("private", false, "mark note as private in frontmatter")
	updateCmd.Flags().Bool("sync-filename", false, "rename the file to match the frontmatter's slug/type cache")
	updateCmd.MarkFlagsMutuallyExclusive("slug", "no-slug")
	updateCmd.MarkFlagsMutuallyExclusive("type", "no-type")
	updateCmd.MarkFlagsMutuallyExclusive("tag", "no-tags")
	updateCmd.MarkFlagsMutuallyExclusive("public", "private")
	rootCmd.AddCommand(updateCmd)
}
