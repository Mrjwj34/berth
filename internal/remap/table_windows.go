//go:build windows

package remap

import "os"

func withTableLock(path string, fn func() error) error {
	if err := os.MkdirAll(dirOf(path), 0o755); err != nil {
		return err
	}
	return fn()
}
