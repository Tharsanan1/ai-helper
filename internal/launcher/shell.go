package launcher

import (
	"fmt"
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

// SetTerminalTitle sets the terminal window title
func SetTerminalTitle(title string) {
	fmt.Fprintf(os.Stderr, "\033]0;%s\007", title)
}
