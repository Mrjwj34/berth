//go:build linux

package remap

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"os/signal"
	"path/filepath"
	"syscall"

	"golang.org/x/sys/unix"
)

// Supervise runs argv under a C wrapper that installs seccomp USER_NOTIF
// and hands the listener fd back. The parent remaps bind/connect so Go
// binaries work; the parent itself is not filtered.
func Supervise(ctx context.Context, argv []string) error {
	if len(argv) == 0 {
		return fmt.Errorf("remap-supervise: missing command")
	}
	if ctx == nil {
		ctx = context.Background()
	}
	bin := argv[0]
	if filepath.Base(bin) == bin {
		if p, err := exec.LookPath(bin); err == nil {
			bin = p
		}
	}
	wrap := WrapPath()
	if _, err := os.Stat(wrap); err != nil {
		fmt.Fprintf(os.Stderr, "lane remap: wrapper missing (%v); libc preload still applies\n", err)
		return runPlain(ctx, bin, append([]string{bin}, argv[1:]...))
	}
	fds, err := unix.Socketpair(unix.AF_UNIX, unix.SOCK_STREAM|unix.SOCK_CLOEXEC, 0)
	if err != nil {
		return fmt.Errorf("remap-supervise socketpair: %w", err)
	}
	wd, err := os.Getwd()
	if err != nil {
		wd = ""
	}
	childArgv := append([]string{wrap, bin}, argv[1:]...)
	env := append(os.Environ(), fmt.Sprintf("LANE_REMAP_COMM_FD=%d", 3))
	pid, err := syscall.ForkExec(wrap, childArgv, &syscall.ProcAttr{
		Dir:   wd,
		Env:   env,
		Files: []uintptr{os.Stdin.Fd(), os.Stdout.Fd(), os.Stderr.Fd(), uintptr(fds[1])},
		Sys:   &syscall.SysProcAttr{Setpgid: true},
	})
	_ = unix.Close(fds[1])
	if err != nil {
		_ = unix.Close(fds[0])
		return fmt.Errorf("remap-supervise start %s: %w", bin, err)
	}
	listener, lerr := recvListener(fds[0])
	_ = unix.Close(fds[0])
	waitErr := make(chan error, 1)
	go func() {
		var ws unix.WaitStatus
		_, err := unix.Wait4(pid, &ws, 0, nil)
		if listener >= 0 {
			_ = unix.Close(listener)
		}
		if err != nil {
			waitErr <- err
			return
		}
		if ws.Signaled() {
			waitErr <- fmt.Errorf("signal: %s", ws.Signal())
			return
		}
		if ws.Exited() && ws.ExitStatus() != 0 {
			waitErr <- fmt.Errorf("exit status %d", ws.ExitStatus())
			return
		}
		waitErr <- nil
	}()
	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGTERM, syscall.SIGINT)
	go func() {
		select {
		case <-ctx.Done():
		case <-sigs:
		}
		_ = unix.Kill(-pid, unix.SIGTERM)
	}()
	if lerr != nil {
		fmt.Fprintf(os.Stderr, "lane remap: seccomp unavailable (%v); libc preload still applies\n", lerr)
		return <-waitErr
	}
	handleLoop(listener, os.Getenv(envFile))
	return <-waitErr
}

func runPlain(ctx context.Context, bin string, argv []string) error {
	cmd := exec.CommandContext(ctx, bin, argv[1:]...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Env = os.Environ()
	if wd, err := os.Getwd(); err == nil {
		cmd.Dir = wd
	}
	return cmd.Run()
}
