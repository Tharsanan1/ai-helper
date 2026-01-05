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

type CopilotLauncher struct {
	cliPath string
}

func NewCopilotLauncher(cliPath string) *CopilotLauncher {
	return &CopilotLauncher{
		cliPath: cliPath,
	}
}

func (c *CopilotLauncher) Name() string {
	return "copilot"
}

func (c *CopilotLauncher) IsAvailable() bool {
	path := c.getCLIPath()
	if path == "" {
		return false
	}

	if _, err := exec.LookPath(path); err != nil {
		return false
	}

	return true
}

func (c *CopilotLauncher) Launch(ctx context.Context, opts LaunchOptions) error {
	if !c.IsAvailable() {
		return pkgerrors.NewToolError("copilot", pkgerrors.ErrToolNotAvailable)
	}

	if opts.Sandbox {
		return c.launchSandbox(ctx, opts)
	}

	path := c.getCLIPath()

	// If NewTerminal is true, open in a new terminal window
	if opts.NewTerminal {
		args := []string{}
		args = append(args, opts.Args...)
		
		// Use simple command name for new terminal to avoid path escaping issues
		command := fmt.Sprintf("copilot %s", joinArgsForShell(args))
		terminalName := opts.TerminalName
		if terminalName == "" {
			terminalName = "Copilot CLI"
		}
		
		return OpenInNewTerminal(opts.WorkDir, command, terminalName)
	}

	args := []string{}
	args = append(args, opts.Args...)

	cmd := exec.CommandContext(ctx, path, args...)
	cmd.Dir = opts.WorkDir
	cmd.Env = os.Environ()

	for k, v := range opts.Env {
		cmd.Env = append(cmd.Env, fmt.Sprintf("%s=%s", k, v))
	}

	if opts.Interactive {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	if err := cmd.Start(); err != nil {
		return fmt.Errorf("failed to start Copilot: %w", err)
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

func (c *CopilotLauncher) getCLIPath() string {
	if c.cliPath != "" {
		return c.cliPath
	}

	if path, err := exec.LookPath("copilot"); err == nil {
		return path
	}

	commonPaths := []string{
		"/usr/local/bin/copilot",
		"/opt/homebrew/bin/copilot",
		"/usr/bin/copilot",
	}

	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}

func (c *CopilotLauncher) launchSandbox(ctx context.Context, opts LaunchOptions) error {
	dockerArgs, err := GetDockerSandboxCommand(opts.WorkDir)
	if err != nil {
		return err
	}

	// Add copilot command and args
	dockerArgs = append(dockerArgs, "copilot", "--allow-all-tools", "--allow-all-paths")
	if len(opts.Args) > 0 {
		dockerArgs = append(dockerArgs, opts.Args...)
	}

	// If NewTerminal is true, open in a new terminal window
	if opts.NewTerminal {
		command := fmt.Sprintf("docker %s", joinArgsForShell(dockerArgs))
		terminalName := opts.TerminalName
		if terminalName == "" {
			terminalName = "Copilot Sandbox"
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
		return fmt.Errorf("failed to start Copilot in sandbox: %w", err)
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
