package launcher

import (
	"fmt"
	"os"
	"os/exec"
	"runtime"
	"strings"

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

// OpenInNewTerminal opens a new terminal window and runs the given command
func OpenInNewTerminal(workDir, command, terminalName string) error {
	switch runtime.GOOS {
	case "darwin":
		return openInMacTerminal(workDir, command, terminalName)
	case "linux":
		return openInLinuxTerminal(workDir, command, terminalName)
	default:
		return fmt.Errorf("opening new terminal not supported on %s", runtime.GOOS)
	}
}

func openInMacTerminal(workDir, command, terminalName string) error {
	// Build the AppleScript to open a new Terminal window
	script := fmt.Sprintf(`
tell application "Terminal"
	set newTab to do script "cd %s && %s"
	set custom title of newTab to "%s"
	activate
end tell
`, escapeForAppleScript(workDir), escapeForAppleScript(command), escapeForAppleScript(terminalName))

	cmd := exec.Command("osascript", "-e", script)
	return cmd.Run()
}

func openInLinuxTerminal(workDir, command, terminalName string) error {
	// Try common Linux terminal emulators in order
	terminals := []struct {
		name string
		args []string
	}{
		{"gnome-terminal", []string{"--", "bash", "-c", fmt.Sprintf("cd %s && %s; exec bash", workDir, command)}},
		{"konsole", []string{"--workdir", workDir, "-e", "bash", "-c", fmt.Sprintf("%s; exec bash", command)}},
		{"xterm", []string{"-e", "bash", "-c", fmt.Sprintf("cd %s && %s; exec bash", workDir, command)}},
	}

	for _, term := range terminals {
		if _, err := exec.LookPath(term.name); err == nil {
			cmd := exec.Command(term.name, term.args...)
			return cmd.Start()
		}
	}

	return fmt.Errorf("no supported terminal emulator found")
}

func escapeForAppleScript(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	return s
}

func joinArgsForShell(args []string) string {
	quoted := make([]string, len(args))
	for i, arg := range args {
		if strings.Contains(arg, " ") {
			quoted[i] = fmt.Sprintf("\"%s\"", arg)
		} else {
			quoted[i] = arg
		}
	}
	return strings.Join(quoted, " ")
}
