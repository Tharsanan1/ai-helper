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

type DroidLauncher struct {
	cliPath string
}

func NewDroidLauncher(cliPath string) *DroidLauncher {
	return &DroidLauncher{
		cliPath: cliPath,
	}
}

func (d *DroidLauncher) Name() string {
	return "droid"
}

func (d *DroidLauncher) IsAvailable() bool {
	path := d.getCLIPath()
	if path == "" {
		return false
	}

	if _, err := exec.LookPath(path); err != nil {
		return false
	}

	return true
}

func (d *DroidLauncher) Launch(ctx context.Context, opts LaunchOptions) error {
	if !d.IsAvailable() {
		return pkgerrors.NewToolError("droid", pkgerrors.ErrToolNotAvailable)
	}

	path := d.getCLIPath()

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
		return fmt.Errorf("failed to start Droid: %w", err)
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

func (d *DroidLauncher) getCLIPath() string {
	if d.cliPath != "" {
		return d.cliPath
	}

	if path, err := exec.LookPath("droid"); err == nil {
		return path
	}

	commonPaths := []string{
		"/usr/local/bin/droid",
		"/opt/homebrew/bin/droid",
		"/usr/bin/droid",
	}

	for _, path := range commonPaths {
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return ""
}
