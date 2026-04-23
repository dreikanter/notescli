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
		n, err := resolveRef(cmd, root, args[0])
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

		contentChanged := false
		if cmd.Flags().Changed("title") {
			updated.Title = updateTitle
			contentChanged = true
		}
		if cmd.Flags().Changed("description") {
			updated.Description = updateDescription
			contentChanged = true
		}
		if updateNoTags {
			updated.Tags = nil
			contentChanged = true
		} else if cmd.Flags().Changed("tag") {
			updated.Tags = updateTags
			contentChanged = true
		}
		if updateNoSlug {
			updated.Slug = ""
			contentChanged = true
		} else if cmd.Flags().Changed("slug") {
			updated.Slug = updateSlug
			contentChanged = true
		}
		if cmd.Flags().Changed("private") {
			v, _ := cmd.Flags().GetBool("private")
			updated.Public = !v
			contentChanged = true
		} else if cmd.Flags().Changed("public") {
			v, _ := cmd.Flags().GetBool("public")
			updated.Public = v
			contentChanged = true
		}
		if updateNoType {
			updated.Type = ""
			contentChanged = true
		} else if cmd.Flags().Changed("type") {
			updated.Type = updateType
			contentChanged = true
		}

		if contentChanged {
			newContent, err := note.FormatNote(updated, body)
			if err != nil {
				return err
			}
			if err := note.WriteAtomic(oldPath, newContent); err != nil {
				return err
			}
		}

		newPath := oldPath
		if syncFilename {
			var syncErr error
			newPath, syncErr = syncNoteFilename(cmd, n, updated, oldPath, updateNoSlug, updateNoType)
			if syncErr != nil {
				return syncErr
			}
		}

		fmt.Fprintln(cmd.OutOrStdout(), newPath)
		return nil
	},
}

// syncNoteFilename reconciles the on-disk filename with the (already-updated)
// frontmatter slug and type. When frontmatter is silent on slug/type and the
// user did not touch those flags, the filename-reported value is used so the
// rename is a no-op instead of stripping a still-valid cache suffix. The rename
// uses hard-link + remove to atomically reserve the target and refuse a clobber.
// Returns newPath, which equals oldPath when no rename is needed.
func syncNoteFilename(cmd *cobra.Command, n note.Ref, updated note.Frontmatter, oldPath string, noSlug, noType bool) (string, error) {
	id, err := strconv.Atoi(n.ID)
	if err != nil {
		return "", fmt.Errorf("invalid note id %q: %w", n.ID, err)
	}
	syncSlug := updated.Slug
	if syncSlug == "" && !cmd.Flags().Changed("slug") && !noSlug {
		syncSlug = n.Slug
	}
	syncType := updated.Type
	if syncType == "" && !cmd.Flags().Changed("type") && !noType {
		syncType = n.Type
	}
	newFilename := note.Filename(n.Date, id, syncSlug, syncType)
	dir := filepath.Dir(oldPath)
	newPath := filepath.Join(dir, newFilename)
	if newPath == oldPath {
		return oldPath, nil
	}
	// os.Link atomically reserves the target: returns EEXIST if it already
	// exists, which os.Rename on Unix would silently clobber.
	if err := os.Link(oldPath, newPath); err != nil {
		if errors.Is(err, os.ErrExist) {
			return "", fmt.Errorf("target note already exists: %s", newPath)
		}
		return "", fmt.Errorf("cannot link note: %w", err)
	}
	if err := os.Remove(oldPath); err != nil {
		// Roll back the link so we don't leave both paths pointing to the
		// same inode. Best-effort: a cleanup failure here isn't surfaced
		// because the original error is more useful.
		_ = os.Remove(newPath)
		return "", fmt.Errorf("cannot remove old note: %w", err)
	}
	return newPath, nil
}

func registerUpdateFlags() {
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
}

func init() {
	registerUpdateFlags()
	rootCmd.AddCommand(updateCmd)
}
