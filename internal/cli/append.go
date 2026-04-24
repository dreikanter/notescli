package cli

import (
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/dreikanter/notes-cli/note"
	"github.com/spf13/cobra"
)

var appendCmd = &cobra.Command{
	Use:   "append <id>",
	Short: "Append text from stdin to a note",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		id, err := strconv.Atoi(args[0])
		if err != nil {
			return fmt.Errorf("id must be an integer: %s", args[0])
		}

		in := cmd.InOrStdin()
		if stdinIsTerminal(in) {
			return fmt.Errorf("no input: pipe text to stdin (e.g. echo 'text' | notes append <id>)")
		}

		data, err := io.ReadAll(in)
		if err != nil {
			return fmt.Errorf("cannot read stdin: %w", err)
		}
		content := strings.TrimSpace(string(data))
		if content == "" {
			return nil
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

		body := entry.Body
		if body != "" && !strings.HasSuffix(body, "\n") {
			body += "\n"
		}
		entry.Body = body + "\n" + content + "\n"

		saved, err := store.Put(entry)
		if err != nil {
			return err
		}

		fmt.Fprintln(cmd.OutOrStdout(), store.AbsPath(saved))
		return nil
	},
}

func init() {
	rootCmd.AddCommand(appendCmd)
}
