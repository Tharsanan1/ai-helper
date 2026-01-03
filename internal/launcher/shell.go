package launcher

import (
	"os"

	"golang.org/x/term"
)

// IsTTY checks if the current session is running in a TTY
func IsTTY() bool {
	return term.IsTerminal(int(os.Stdout.Fd()))
}

// GetShell returns the user's shell or a default
func GetShell() string {
	shell := os.Getenv("SHELL")
	if shell == "" {
		shell = "/bin/sh"
	}
	return shell
}
