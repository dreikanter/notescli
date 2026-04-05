package cli

import (
	"fmt"
	"os"
	"path/filepath"
	"time"

	"github.com/dreikanter/notescli/note"
	"github.com/spf13/cobra"
)

var rmCmd = &cobra.Command{
	Use:   "rm <id|path|basename|slug|type>",
	Short: "Delete a note",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		today, _ := cmd.Flags().GetBool("today")

		root := mustNotesPath()

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

func init() {
	rmCmd.Flags().Bool("today", false, "only match notes created today")
	rootCmd.AddCommand(rmCmd)
}
