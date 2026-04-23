package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dreikanter/notes-cli/note"
	"github.com/spf13/cobra"
)

var newTodoCmd = &cobra.Command{
	Use:   "new-todo",
	Short: "Create today's todo",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := notesRoot()
		if err != nil {
			return err
		}
		today := time.Now().Format(note.DateFormat)

		idx, err := note.Load(root, note.WithFrontmatter(false), note.WithLogger(stderrLogger(cmd)))
		if err != nil {
			return err
		}
		entries := idx.Entries()

		if existing := note.FindTodayTodo(entries, today); existing != nil {
			fmt.Fprintln(cmd.OutOrStdout(), filepath.Join(root, existing.RelPath))
			return nil
		}

		// Find the most recent previous todo and roll over tasks
		var carriedTasks []note.Task
		prev := note.FindLatestTodo(entries, today)
		if prev != nil {
			prevPath := filepath.Join(root, prev.RelPath)
			prevData, err := os.ReadFile(prevPath)
			if err != nil {
				return fmt.Errorf("cannot read previous todo: %w", err)
			}
			prevLines := strings.Split(string(prevData), "\n")

			result := note.RolloverTasks(prevLines)
			carriedTasks = result.CarriedTasks

			if err := note.WriteAtomic(prevPath, []byte(strings.Join(result.UpdatedLines, "\n"))); err != nil {
				return fmt.Errorf("cannot update previous todo: %w", err)
			}
		}

		// Allocate new ID and create new todo
		id, err := note.NextID(root)
		if err != nil {
			return err
		}

		filename := note.Filename(today, id, "", "todo")
		dir := note.DirPath(root, today)
		if err := os.MkdirAll(dir, note.StoreDirMode(root)); err != nil {
			return fmt.Errorf("cannot create directory %s: %w", dir, err)
		}

		fullPath := filepath.Join(dir, filename)
		body := []byte(note.FormatTodoContent(carriedTasks))
		content, err := note.FormatNote(note.Frontmatter{Type: "todo"}, body)
		if err != nil {
			return err
		}

		if err := note.WriteAtomic(fullPath, content); err != nil {
			return err
		}

		fmt.Fprintln(cmd.OutOrStdout(), fullPath)
		return nil
	},
}

func init() {
	rootCmd.AddCommand(newTodoCmd)
}
