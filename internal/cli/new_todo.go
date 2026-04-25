package cli

import (
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/dreikanter/notesctl/note"
	"github.com/spf13/cobra"
)

var newTodoCmd = &cobra.Command{
	Use:   "new-todo",
	Short: "Create today's todo, carrying over incomplete tasks from the previous todo",
	Args:  cobra.NoArgs,
	RunE: func(cmd *cobra.Command, args []string) error {
		store, err := notesStore()
		if err != nil {
			return err
		}
		today := time.Now()

		if existing, err := store.Find(note.WithType("todo"), note.WithExactDate(today)); err == nil {
			fmt.Fprintln(cmd.OutOrStdout(), store.AbsPath(existing))
			return nil
		} else if !errors.Is(err, note.ErrNotFound) {
			return err
		}

		var carriedTasks []note.Task
		prev, err := store.Find(note.WithType("todo"), note.WithBeforeDate(today))
		switch {
		case err == nil:
			prevLines := strings.Split(prev.Body, "\n")
			result := note.RolloverTasks(prevLines)
			carriedTasks = result.CarriedTasks

			prev.Body = strings.Join(result.UpdatedLines, "\n")
			if _, err := store.Put(prev); err != nil {
				return fmt.Errorf("cannot update previous todo: %w", err)
			}
		case errors.Is(err, note.ErrNotFound):
			// no previous todo; carriedTasks stays empty
		default:
			return err
		}

		saved, err := store.Put(note.Entry{
			Meta: note.Meta{Type: "todo", CreatedAt: today},
			Body: note.FormatTodoContent(carriedTasks),
		})
		if err != nil {
			return err
		}

		fmt.Fprintln(cmd.OutOrStdout(), store.AbsPath(saved))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(newTodoCmd)
}
