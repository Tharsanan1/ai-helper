package launcher

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"

	pkgerrors "github.com/tharsanan1/ai-helper/pkg/errors"
)

// ClaudeLauncher implements ToolLauncher for Claude Code CLI
type ClaudeLauncher struct {
	cliPath string
}

// NewClaudeLauncher creates a new Claude launcher
// If cliPath is empty, it will be auto-detected
func NewClaudeLauncher(cliPath string) *ClaudeLauncher {
	return &ClaudeLauncher{
		cliPath: cliPath,
	}
}

// Name returns the name of the tool
func (c *ClaudeLauncher) Name() string {
	return "claude"
}

// IsAvailable checks if Claude CLI is available
func (c *ClaudeLauncher) IsAvailable() bool {
	path := c.getCLIPath()
	if path == "" {
		return false
	}

	// Check if the binary exists and is executable
	if _, err := exec.LookPath(path); err != nil {
		return false
	}

	return true
}

// Launch launches Claude CLI with the given options
func (c *ClaudeLauncher) Launch(ctx context.Context, opts LaunchOptions) error {
	if !c.IsAvailable() {
		return pkgerrors.NewToolError("claude", pkgerrors.ErrToolNotAvailable)
	}

	if opts.Sandbox {
		return c.launchSandbox(ctx, opts)
	}

	path := c.getCLIPath()

	// Build command arguments
	args := []string{}

	// Add mode if specified
	if opts.Mode != "" {
		switch opts.Mode {
		case "agent", "chat":
			// Claude accepts mode as a flag or can be inferred
			// For now, we'll just pass it to the working directory
			// The user can configure their Claude to default to a mode
		default:
			return fmt.Errorf("invalid Claude mode: %s (must be 'agent' or 'chat')", opts.Mode)
		}
	}

	// Add additional arguments
	args = append(args, opts.Args...)

	// If NewTerminal is true, open in a new terminal window
	if opts.NewTerminal {
		command := fmt.Sprintf("claude %s", joinArgsForShell(args))
		terminalName := opts.TerminalName
		if terminalName == "" {
			terminalName = "Claude CLI"
		}
		
		return OpenInNewTerminal(opts.WorkDir, command, terminalName)
	}

	// Create command
	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Dir = opts.WorkDir
	cmd.Env = os.Environ()

	// Add custom environment variables
	for k, v := range opts.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	// Set up input/output
	if opts.Interactive {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	// Handle signals for graceful shutdown
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	// Start the command
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Claude: %w", err)
	}

	// Set terminal name if specified
	if opts.TerminalName != "" && opts.Interactive {
		SetTerminalTitle(opts.TerminalName)
	}

	// Wait for command to finish or signal
	errChan := make(chan error, 1)
	go func() {
		errChan <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		// Context cancelled, kill the process
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		return ctx.Err()
	case sig := <-sigChan:
		// Signal received, forward to child process
		if cmd.Process != nil {
			_ = cmd.Process.Signal(sig)
		}
		// Wait a bit for graceful shutdown
		<-errChan
		return nil
	case err := <-errChan:
		// Command finished
		return err
	}
}

// getCLIPath returns the path to Claude CLI
func (c *ClaudeLauncher) getCLIPath() string {
	if c.cliPath != "" {
		return c.cliPath
	}

	// Try to find in PATH
	if path, err := exec.LookPath("claude"); err == nil {
		return path
	}

	// Try common installation locations
	commonPaths := []string{
		"/usr/local/bin/claude",
		"/opt/homebrew/bin/claude",
		"/usr/bin/claude",
	}

	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

func (c *ClaudeLauncher) launchSandbox(ctx context.Context, opts LaunchOptions) error {
	dockerArgs, err := GetDockerSandboxCommand(opts.WorkDir)
	if err != nil {
		return err
	}

	// Add claude command and args
	dockerArgs = append(dockerArgs, "claude", "--dangerously-skip-permissions")
	
	// Add mode if specified (though for sandbox we might just rely on args?)
	// The original Launch method had logic for mode.
	if opts.Mode != "" {
		// If mode is passed as argument, append it?
		// But in docker we are just running "claude ..."
		// If users pass "agent" mode, does it map to an arg? 
		// "claude agent" or "claude chat"
		// The original code said: "Claude accepts mode as a flag or can be inferred... For now, we'll just pass it to the working directory"
		// Wait, the original code for mode did NOTHING with args:
		/*
		case "agent", "chat":
			// Claude accepts mode as a flag or can be inferred
			// For now, we'll just pass it to the working directory
			// The user can configure their Claude to default to a mode
		*/
		// It just validated it.
	}

	if len(opts.Args) > 0 {
		dockerArgs = append(dockerArgs, opts.Args...)
	}

	// If NewTerminal is true, open in a new terminal window
	if opts.NewTerminal {
		command := fmt.Sprintf("docker %s", joinArgsForShell(dockerArgs))
		terminalName := opts.TerminalName
		if terminalName == "" {
			terminalName = "Claude Sandbox"
		}
		
		return OpenInNewTerminal(opts.WorkDir, command, terminalName)
	}

	cmd := exec.CommandContext(ctx, "docker", dockerArgs...)
	cmd.Env = os.Environ()

	if opts.Interactive {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Claude in sandbox: %w", err)
	}

	if opts.TerminalName != "" && opts.Interactive {
		SetTerminalTitle(opts.TerminalName)
	}

	errChan := make(chan error, 1)
	go func() {
		errChan <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		return ctx.Err()
	case sig := <-sigChan:
		if cmd.Process != nil {
			_ = cmd.Process.Signal(sig)
		}
		<-errChan
		return nil
	case err := <-errChan:
		return err
	}
}
