package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime/debug"

	"github.com/spf13/cobra"
)

var (
	notesPath string
	Version   = "dev"
)

var rootCmd = &cobra.Command{
	Use:          "notes",
	Short:        "Interact with a notes store",
	SilenceUsage: true,
}

func init() {
	if Version == "dev" {
		if info, ok := debug.ReadBuildInfo(); ok && info.Main.Version != "(devel)" {
			Version = info.Main.Version
		}
	}
	rootCmd.Version = Version
	rootCmd.PersistentFlags().StringVar(&notesPath, "path", "", "path to notes store (default: $NOTES_PATH)")
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
	return "", errors.New("no notes store configured. Set $NOTES_PATH or pass --path")
}

func mustNotesPath() string {
	p, err := resolveNotesPath()
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	p, err = filepath.Abs(p)
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
