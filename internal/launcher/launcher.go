package launcher

import "context"

// ToolLauncher is an interface for launching development tools
type ToolLauncher interface {
	// Launch launches the tool with the given options
	Launch(ctx context.Context, opts LaunchOptions) error

	// IsAvailable checks if the tool is available on the system
	IsAvailable() bool

	// Name returns the name of the tool
	Name() string
}

// LaunchOptions contains options for launching a tool
type LaunchOptions struct {
	// WorkDir is the working directory for the tool
	WorkDir string

	// Args are additional arguments to pass to the tool
	Args []string

	// Mode is the mode to launch the tool in (tool-specific)
	Mode string

	// Env is additional environment variables to set
	Env map[string]string

	// Interactive determines if the tool should run interactively
	Interactive bool

	// TerminalName is the name to set for the terminal window
	TerminalName string

	// NewTerminal determines if the tool should launch in a new terminal window
	NewTerminal bool

	// Sandbox determines if the tool should run in a docker sandbox
	Sandbox bool
}
