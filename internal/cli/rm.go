package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dreikanter/notes-cli/note"
	"github.com/spf13/cobra"
)

var rmCmd = &cobra.Command{
	Use:   "rm <id|type|query>",
	Short: "Delete a note",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		today, _ := cmd.Flags().GetBool("today")

		root, err := notesRoot()
		if err != nil {
			return err
		}

		var date string
		if today {
			date = time.Now().Format("20060102")
		}

		n, err := note.ResolveRefDate(root, args[0], date)
		if err != nil {
			return err
		}

		absPath := filepath.Join(root, n.RelPath)
		if err := os.Remove(absPath); err != nil {
			return err
		}

		fmt.Fprintln(cmd.OutOrStdout(), absPath)
		return nil
	},
}

func registerRmFlags() {
	rmCmd.Flags().Bool("today", false, "only match notes created today")
}

func init() {
	registerRmFlags()
	rootCmd.AddCommand(rmCmd)
}
