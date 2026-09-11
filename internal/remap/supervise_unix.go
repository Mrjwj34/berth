//go:build unix

package remap

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"syscall"
	"time"
)

// Spawn starts `lane remap-supervise -- <bin> <args...>` in a new session.
func Spawn(ctx context.Context, worktree, bin string, args, env []string) error {
	exe, err := os.Executable()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(dirOf(LogPath(worktree)), 0o755); err != nil {
		return err
	}
	logf, err := os.OpenFile(LogPath(worktree), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	cmdArgs := append([]string{"remap-supervise", "--", bin}, args...)
	cmd := exec.Command(exe, cmdArgs...)
	cmd.Dir = worktree
	cmd.Env = env
	cmd.Stdout = logf
	cmd.Stderr = logf
	cmd.SysProcAttr = &syscall.SysProcAttr{Setsid: true}
	if err := cmd.Start(); err != nil {
		_ = logf.Close()
		return fmt.Errorf("start remap supervisor: %w", err)
	}
	if err := os.WriteFile(PIDPath(worktree), []byte(fmt.Sprintf("%d\n", cmd.Process.Pid)), 0o644); err != nil {
		_ = cmd.Process.Kill()
		_ = logf.Close()
		return err
	}
	go func() {
		_ = cmd.Wait()
		_ = logf.Close()
	}()
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if err := ctx.Err(); err != nil {
			return err
		}
		if processAlive(cmd.Process.Pid) {
			return nil
		}
		time.Sleep(20 * time.Millisecond)
	}
	return nil
}

// Stop kills a leftover remap supervisor for the workspace.
func Stop(worktree string) error {
	pid := readPID(PIDPath(worktree))
	_ = os.Remove(PIDPath(worktree))
	if pid <= 0 || !processAlive(pid) {
		return nil
	}
	_ = syscall.Kill(pid, syscall.SIGTERM)
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if !processAlive(pid) {
			return nil
		}
		time.Sleep(20 * time.Millisecond)
	}
	_ = syscall.Kill(pid, syscall.SIGKILL)
	return nil
}

func processAlive(pid int) bool {
	if pid <= 0 {
		return false
	}
	return syscall.Kill(pid, 0) == nil
}
