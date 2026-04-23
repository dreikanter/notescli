package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/dreikanter/notes-cli/note"
	"github.com/spf13/cobra"
)

var newCmd = &cobra.Command{
	Use:   "new",
	Short: "Create a new note",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		slug, _ := cmd.Flags().GetString("slug")
		noteType, _ := cmd.Flags().GetString("type")
		tags, _ := cmd.Flags().GetStringSlice("tag")
		description, _ := cmd.Flags().GetString("description")
		title, _ := cmd.Flags().GetString("title")
		publicFlag, _ := cmd.Flags().GetBool("public")
		upsert, _ := cmd.Flags().GetBool("upsert")

		if err := note.ValidateSlug(slug); err != nil {
			return err
		}

		if upsert && noteType == "" && slug == "" {
			return fmt.Errorf("--upsert requires --type or --slug")
		}

		root, err := notesRoot()
		if err != nil {
			return err
		}

		if upsert {
			path, found, err := findUpsertNote(cmd, root, noteType, slug)
			if err != nil {
				return err
			}
			if found {
				fmt.Fprintln(cmd.OutOrStdout(), path)
				return nil
			}
		}

		body, err := readStdinBody(cmd)
		if err != nil {
			return err
		}

		fullPath, err := createNote(createNoteParams{
			Root:        root,
			Slug:        slug,
			Type:        noteType,
			Tags:        tags,
			Title:       title,
			Description: description,
			Public:      publicFlag,
			Body:        body,
		})
		if err != nil {
			return err
		}

		fmt.Fprintln(cmd.OutOrStdout(), fullPath)
		return nil
	},
}

// stdinIsTerminal reports whether in looks like an interactive terminal. Only
// *os.File readers are heuristically inspected; any other reader (a pipe,
// buffer, or other io.Reader injected via cmd.SetIn) is treated as non-terminal
// so tests and piped invocations read the provided bytes.
func stdinIsTerminal(in io.Reader) bool {
	f, ok := in.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return true
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// findUpsertNote returns the absolute path of an existing today's note
// matching noteType and slug, or ("", false, nil) when none exists.
func findUpsertNote(cmd *cobra.Command, root, noteType, slug string) (string, bool, error) {
	today := time.Now().Format(note.DateFormat)
	idx, err := note.Load(root, note.WithFrontmatter(false), note.WithLogger(stderrLogger(cmd)))
	if err != nil {
		return "", false, err
	}
	entries := note.FilterByDate(idx.Entries(), today)
	if noteType != "" {
		entries = note.FilterByTypes(entries, []string{noteType})
	}
	if slug != "" {
		entries = note.FilterBySlug(entries, slug)
	}
	if len(entries) == 0 {
		return "", false, nil
	}
	return filepath.Join(root, entries[0].RelPath), true, nil
}

// readStdinBody reads stdin when it is not a terminal and returns its content.
// Returns ("", nil) when stdin is a terminal (no piped input).
func readStdinBody(cmd *cobra.Command) (string, error) {
	in := cmd.InOrStdin()
	if stdinIsTerminal(in) {
		return "", nil
	}
	data, err := io.ReadAll(in)
	if err != nil {
		return "", fmt.Errorf("cannot read stdin: %w", err)
	}
	return string(data), nil
}

func registerNewFlags() {
	newCmd.Flags().String("slug", "", "descriptive slug appended to filename")
	newCmd.Flags().String("type", "", "note type (free-form; todo/backlog/weekly get special behavior)")
	newCmd.Flags().StringSlice("tag", nil, "tag for frontmatter (repeatable)")
	newCmd.Flags().String("description", "", "description for frontmatter")
	newCmd.Flags().String("title", "", "title for frontmatter")
	newCmd.Flags().Bool("public", false, "mark note as public in frontmatter")
	newCmd.Flags().Bool("private", false, "mark note as private in frontmatter (default)")
	newCmd.Flags().Bool("upsert", false, "return existing note if today already has one matching --type/--slug")
	newCmd.MarkFlagsMutuallyExclusive("public", "private")
}

func init() {
	registerNewFlags()
	rootCmd.AddCommand(newCmd)
}
