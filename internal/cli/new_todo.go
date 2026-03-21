package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/dreikanter/notescli/note"
	"github.com/spf13/cobra"
)

var newTodoForce bool

var newTodoCmd = &cobra.Command{
	Use:   "new-todo",
	Short: "Create today's todo from the previous todo",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		root := mustNotesPath()
		today := time.Now().Format("20060102")

		notes, err := note.Scan(root)
		if err != nil {
			return err
		}

		// Check if today's todo already exists
		if !newTodoForce {
			if existing := note.FindTodayTodo(notes, today); existing != nil {
				fmt.Println(filepath.Join(root, existing.RelPath))
				return nil
			}
		}

		// Find the most recent previous todo
		prev := note.FindLatestTodo(notes, today)
		if prev == nil {
			return fmt.Errorf("no previous todo found")
		}

		// Read previous todo content
		prevPath := filepath.Join(root, prev.RelPath)
		prevData, err := os.ReadFile(prevPath)
		if err != nil {
			return fmt.Errorf("cannot read previous todo: %w", err)
		}
		prevLines := strings.Split(string(prevData), "\n")

		// Rollover tasks
		result := note.RolloverTasks(prevLines, newTodoForce)

		// Write back modified previous todo
		if err := os.WriteFile(prevPath, []byte(strings.Join(result.UpdatedLines, "\n")), 0o644); err != nil {
			return fmt.Errorf("cannot update previous todo: %w", err)
		}

		// Allocate new ID and create new todo
		id, err := note.NextID(root)
		if err != nil {
			return err
		}

		filename := note.NoteFilename(today, id, "todo")
		dir := note.NoteDirPath(root, today)
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return fmt.Errorf("cannot create directory %s: %w", dir, err)
		}

		fullPath := filepath.Join(dir, filename)
		content := note.FormatTodoContent(result.CarriedTasks)

		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			return fmt.Errorf("cannot write todo: %w", err)
		}

		fmt.Println(fullPath)
		return nil
	},
}

func init() {
	newTodoCmd.Flags().BoolVar(&newTodoForce, "force", false, "regenerate today's todo even if it exists, and carry over in-progress tasks")
	rootCmd.AddCommand(newTodoCmd)
}
