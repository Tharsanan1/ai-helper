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

type GeminiLauncher struct {
	cliPath string
}

func NewGeminiLauncher(cliPath string) *GeminiLauncher {
	return &GeminiLauncher{
		cliPath: cliPath,
	}
}

func (g *GeminiLauncher) Name() string {
	return "gemini"
}

func (g *GeminiLauncher) IsAvailable() bool {
	path := g.getCLIPath()
	if path == "" {
		return false
	}

	if _, err := exec.LookPath(path); err != nil {
		return false
	}

	return true
}

func (g *GeminiLauncher) Launch(ctx context.Context, opts LaunchOptions) error {
	if !g.IsAvailable() {
		return pkgerrors.NewToolError("gemini", pkgerrors.ErrToolNotAvailable)
	}

	path := g.getCLIPath()

	args := []string{}
	args = append(args, opts.Args...)

	// If NewTerminal is true, open in a new terminal window
	if opts.NewTerminal {
		command := fmt.Sprintf("gemini %s", joinArgsForShell(args))
		terminalName := opts.TerminalName
		if terminalName == "" {
			terminalName = "Gemini CLI"
		}
		
		return OpenInNewTerminal(opts.WorkDir, command, terminalName)
	}

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
		return fmt.Errorf("failed to start Gemini: %w", err)
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

func (g *GeminiLauncher) getCLIPath() string {
	if g.cliPath != "" {
		return g.cliPath
	}

	if path, err := exec.LookPath("gemini"); err == nil {
		return path
	}

	commonPaths := []string{
		"/usr/local/bin/gemini",
		"/opt/homebrew/bin/gemini",
		"/usr/bin/gemini",
	}

	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}
