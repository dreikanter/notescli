package cli

import (
	"fmt"
	"io"
	"os"

	"github.com/spf13/cobra"
)

// stdinIsTerminal reports whether in looks like an interactive terminal. Only
// *os.File readers are heuristically inspected; any other reader (a pipe,
// buffer, or other io.Reader injected via cmd.SetIn) is treated as non-terminal
// so tests and piped invocations read the provided bytes.
func stdinIsTerminal(in io.Reader) bool {
	f, ok := in.(*os.File)
	if !ok {
		return false
	}
	fi, err := f.Stat()
	if err != nil {
		return true
	}
	return fi.Mode()&os.ModeCharDevice != 0
}

// readStdinBody reads stdin when it is not a terminal and returns its content.
// Returns ("", nil) when stdin is a terminal (no piped input).
func readStdinBody(cmd *cobra.Command) (string, error) {
	in := cmd.InOrStdin()
	if stdinIsTerminal(in) {
		return "", nil
	}
	data, err := io.ReadAll(in)
	if err != nil {
		return "", fmt.Errorf("cannot read stdin: %w", err)
	}
	return string(data), nil
}
