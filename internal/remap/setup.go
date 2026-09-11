package remap

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

var errUnsupported = errors.New("isolate: net remaps bind/connect on the host network so hardcoded listen ports can run in parallel without blocking public internet. This is not implemented on Windows. Use Linux or macOS, run a single workspace, or pass the process a unique --port $LANE_PORT_*")

// Setup compiles the interceptor and seeds the workspace remap table.
func Setup(ctx context.Context, worktree string, maps []Mapping, reserved []int) error {
	if runtime.GOOS == "windows" {
		return errUnsupported
	}
	if _, err := EnsureLib(ctx); err != nil {
		return fmt.Errorf("prepare listen remap: %w", err)
	}
	if err := os.MkdirAll(dirOf(TablePath(worktree)), 0o755); err != nil {
		return err
	}
	return seedTable(TablePath(worktree), maps, reserved)
}

func Unsupported() error { return errUnsupported }

// ShouldSupervise reports whether process-compose should run under the
// Linux seccomp helper (needed for Go binaries that skip libc bind/connect).
func ShouldSupervise() bool {
	if runtime.GOOS != "linux" {
		return false
	}
	exe, err := os.Executable()
	if err != nil {
		return false
	}
	return filepath.Base(exe) == "lane"
}

func readPID(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	n, _ := strconv.Atoi(strings.TrimSpace(string(data)))
	return n
}
