package cli

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"
)

var (
	notesPath string
	Version   = "0.4.0"
)

var rootCmd = &cobra.Command{
	Use:          "notes",
	Short:        "Interact with a notes archive",
	Long:         "A CLI tool for reading, filtering, and listing notes in a date-based archive.",
	SilenceUsage: true,
}

func init() {
	rootCmd.Version = Version
	rootCmd.PersistentFlags().StringVar(&notesPath, "path", "", "path to notes archive (overrides NOTES_PATH env var)")
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func resolveNotesPath() (string, error) {
	if notesPath != "" {
		return notesPath, nil
	}
	if env := os.Getenv("NOTES_PATH"); env != "" {
		return env, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("cannot determine home directory: %w", err)
	}
	return filepath.Join(home, "Dropbox", "Notes"), nil
}

func mustNotesPath() string {
	p, err := resolveNotesPath()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	info, err := os.Stat(p)
	if err != nil || !info.IsDir() {
		fmt.Fprintf(os.Stderr, "notes path does not exist or is not a directory: %s\n", p)
		os.Exit(1)
	}

	return p
}
