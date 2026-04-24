package cli

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime/debug"

	"github.com/dreikanter/notes-cli/note"
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
		var ee *exec.ExitError
		if errors.As(err, &ee) {
			os.Exit(ee.ExitCode())
		}
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

func notesRoot() (string, error) {
	p, err := resolveNotesPath()
	if err != nil {
		return "", err
	}

	p, err = filepath.Abs(p)
	if err != nil {
		return "", err
	}

	info, err := os.Stat(p)
	if err != nil || !info.IsDir() {
		return "", fmt.Errorf("notes path does not exist or is not a directory: %s", p)
	}

	return p, nil
}

// notesStore returns the Store instance the CLI uses for note-package
// operations. It resolves the root path the same way notesRoot does — flag
// first, then $NOTES_PATH, then error — so commands can replace their
// direct filesystem calls with Store calls without touching path-resolution
// code.
//
// Phase 3 wires this helper in; individual commands adopt it in later
// phases.
func notesStore() (*note.OSStore, error) {
	root, err := notesRoot()
	if err != nil {
		return nil, err
	}
	return note.NewOSStore(root), nil
}

// Keep notesStore in reach of the unused linter until Phase 4 wires the
// first command through it.
var _ = notesStore
