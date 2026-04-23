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
	today := time.Now().Format(note.DateFormat)

	id, err := note.NextID(p.Root)
	if err != nil {
		return "", err
	}

	filename := note.Filename(today, id, p.Slug, p.Type)
	dir := note.DirPath(p.Root, today)

	if err := os.MkdirAll(dir, note.StoreDirMode(p.Root)); err != nil {
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
	content, err := note.FormatNote(fm, []byte(p.Body))
	if err != nil {
		return "", err
	}

	if err := note.WriteAtomic(fullPath, content); err != nil {
		return "", err
	}

	return fullPath, nil
}
