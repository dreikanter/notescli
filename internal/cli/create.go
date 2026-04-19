package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dreikanter/notes-cli/note"
)

// createNoteParams holds the parameters for creating a new note file.
type createNoteParams struct {
	Root        string
	Slug        string
	Type        string
	Tags        []string
	Title       string
	Description string
	Public      bool
	Body        string // initial content after frontmatter
}

// createNote creates a new note file with optional frontmatter and body content.
// Returns the absolute path to the created file.
func createNote(p createNoteParams) (string, error) {
	today := time.Now().Format("20060102")

	id, err := note.NextID(p.Root)
	if err != nil {
		return "", err
	}

	filename := note.NoteFilename(today, id, p.Slug, p.Type)
	dir := note.NoteDirPath(p.Root, today)

	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", fmt.Errorf("cannot create directory %s: %w", dir, err)
	}

	fullPath := filepath.Join(dir, filename)

	fm := note.Frontmatter{
		Title:       p.Title,
		Slug:        p.Slug,
		Type:        p.Type,
		Tags:        p.Tags,
		Description: p.Description,
		Public:      p.Public,
	}
	content := note.FormatNote(fm, []byte(p.Body))

	if err := os.WriteFile(fullPath, content, 0o644); err != nil {
		return "", fmt.Errorf("cannot write note: %w", err)
	}

	return fullPath, nil
}
