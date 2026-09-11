//go:build darwin

package remap

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

// Supervise runs argv with DYLD remap already in the environment.
func Supervise(ctx context.Context, argv []string) error {
	if len(argv) == 0 {
		return fmt.Errorf("remap-supervise: missing command")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	if wd, err := os.Getwd(); err == nil {
		cmd.Dir = wd
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	if err := cmd.Start(); err != nil {
		return fmt.Errorf("remap-supervise start %s: %w", argv[0], err)
	}
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		select {
		case <-ctx.Done():
		case <-sigs:
		}
		if cmd.Process != nil {
			_ = syscall.Kill(-cmd.Process.Pid, syscall.SIGTERM)
		}
	}()
	return cmd.Wait()
}
