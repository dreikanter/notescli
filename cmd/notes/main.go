package main

import (
	"os/signal"
	"syscall"

	"github.com/dreikanter/notescli/internal/cli"
)

func main() {
	// Let the OS terminate the process on SIGPIPE (conventional
	// behavior for CLI tools piped through head, less, etc.).
	signal.Reset(syscall.SIGPIPE)
	cli.Execute()
}
