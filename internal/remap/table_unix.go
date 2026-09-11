//go:build unix

package remap

import (
	"fmt"
	"os"
	"syscall"
)

func withTableLock(path string, fn func() error) error {
	if err := os.MkdirAll(dirOf(path), 0o755); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	defer f.Close()
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		return fmt.Errorf("lock remap table: %w", err)
	}
	defer func() { _ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN) }()
	return fn()
}
