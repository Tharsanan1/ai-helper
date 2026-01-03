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

type OpenCodeLauncher struct {
	cliPath string
}

func NewOpenCodeLauncher(cliPath string) *OpenCodeLauncher {
	return &OpenCodeLauncher{
		cliPath: cliPath,
	}
}

func (o *OpenCodeLauncher) Name() string {
	return "opencode"
}

func (o *OpenCodeLauncher) IsAvailable() bool {
	path := o.getCLIPath()
	if path == "" {
		return false
	}

	if _, err := exec.LookPath(path); err != nil {
		return false
	}

	return true
}

func (o *OpenCodeLauncher) Launch(ctx context.Context, opts LaunchOptions) error {
	if !o.IsAvailable() {
		return pkgerrors.NewToolError("opencode", pkgerrors.ErrToolNotAvailable)
	}

	path := o.getCLIPath()

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
		return fmt.Errorf("failed to start OpenCode: %w", err)
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

func (o *OpenCodeLauncher) getCLIPath() string {
	if o.cliPath != "" {
		return o.cliPath
	}

	if path, err := exec.LookPath("opencode"); err == nil {
		return path
	}

	commonPaths := []string{
		"/usr/local/bin/opencode",
		"/opt/homebrew/bin/opencode",
		"/usr/bin/opencode",
	}

	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}
