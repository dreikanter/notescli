package cli

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/dreikanter/notes-cli/note"
	"github.com/spf13/cobra"
)

var appendCmd = &cobra.Command{
	Use:   "append [<id|type|query>]",
	Short: "Append text from stdin to a note",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		root, err := notesRoot()
		if err != nil {
			return err
		}

		// Check stdin is piped
		if isTerminal(os.Stdin) {
			return fmt.Errorf("no input: pipe text to stdin (e.g. echo 'text' | notes append <target>)")
		}

		// Read and trim stdin
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("cannot read stdin: %w", err)
		}
		content := strings.TrimSpace(string(data))
		if content == "" {
			return nil
		}

		f := readFilterFlags(cmd)

		var targetPath string

		if len(args) == 1 {
			if f.active() {
				return fmt.Errorf("cannot combine positional argument with filter flags")
			}

			n, resolveErr := note.ResolveRef(root, args[0])
			if resolveErr != nil {
				return resolveErr
			}
			targetPath = filepath.Join(root, n.RelPath)
		} else if f.active() {
			idx, loadErr := note.Load(root, loadOptsFor(f))
			if loadErr != nil {
				return loadErr
			}

			entries := applyFilters(idx.Entries(), f)

			if len(entries) == 0 {
				return fmt.Errorf("no notes found matching filters: %s", f.describe())
			}
			targetPath = filepath.Join(root, entries[0].RelPath)
		} else {
			return fmt.Errorf("specify a note by positional argument or filter flags (--type, --slug, --tag, --today)")
		}

		// Read existing file
		existing, err := os.ReadFile(targetPath)
		if err != nil {
			return fmt.Errorf("cannot read note: %w", err)
		}

		// Append: ensure existing ends with \n, then \n + content + \n
		existingStr := string(existing)
		if len(existingStr) > 0 && !strings.HasSuffix(existingStr, "\n") {
			existingStr += "\n"
		}
		result := existingStr + "\n" + content + "\n"

		if err := writeAtomic(targetPath, []byte(result)); err != nil {
			return err
		}

		fmt.Fprintln(cmd.OutOrStdout(), targetPath)
		return nil
	},
}

func registerAppendFlags() {
	addFilterFlags(appendCmd)
}

func init() {
	registerAppendFlags()
	rootCmd.AddCommand(appendCmd)
}
