package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/dreikanter/notescli/note"
	"github.com/spf13/cobra"
)

var updateCmd = &cobra.Command{
	Use:   "update <ref>",
	Short: "Update frontmatter and/or rename a note",
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

		if updateType != "" && !note.IsKnownType(updateType) {
			return fmt.Errorf("unknown note type %q (valid types: %s)", updateType, strings.Join(note.KnownTypes, ", "))
		}

		root := mustNotesPath()
		n, err := note.ResolveRef(root, args[0])
		if err != nil {
			return err
		}

		oldPath := filepath.Join(root, n.RelPath)
		data, err := os.ReadFile(oldPath)
		if err != nil {
			return fmt.Errorf("cannot read note: %w", err)
		}

		existing := note.ParseFrontmatterFields(data)
		body := note.StripFrontmatter(data)

		// Merge frontmatter updates.
		updated := existing

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

		// Determine new slug.
		newSlug := n.Slug
		if updateNoSlug {
			newSlug = ""
		} else if cmd.Flags().Changed("slug") {
			newSlug = updateSlug
		}
		if updateNoSlug || cmd.Flags().Changed("slug") {
			updated.Slug = newSlug
		}

		// Determine new type.
		newType := n.Type
		if updateNoType {
			newType = ""
		} else if cmd.Flags().Changed("type") {
			newType = updateType
		}

		// n.ID is guaranteed to be a non-empty digit string by ParseFilename.
		id, _ := strconv.Atoi(n.ID)

		newFilename := note.NoteFilename(n.Date, id, newSlug, newType)
		dir := filepath.Dir(oldPath)
		newPath := filepath.Join(dir, newFilename)

		newContent := note.BuildFrontmatter(updated) + string(body)

		tmpPath := newPath + ".tmp"
		if err := os.WriteFile(tmpPath, []byte(newContent), 0o644); err != nil {
			return fmt.Errorf("cannot write note: %w", err)
		}
		if err := os.Rename(tmpPath, newPath); err != nil {
			os.Remove(tmpPath)
			return fmt.Errorf("cannot rename note: %w", err)
		}
		if newPath != oldPath {
			if err := os.Remove(oldPath); err != nil {
				return fmt.Errorf("cannot remove old note: %w", err)
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
	updateCmd.Flags().String("slug", "", "update slug and rename file")
	updateCmd.Flags().Bool("no-slug", false, "remove slug from filename")
	updateCmd.Flags().String("type", "", "update note type and rename file (todo, backlog, weekly)")
	updateCmd.Flags().Bool("no-type", false, "remove type suffix from filename")
	rootCmd.AddCommand(updateCmd)
}
