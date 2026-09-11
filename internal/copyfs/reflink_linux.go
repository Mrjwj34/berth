//go:build linux

package copyfs

import (
	"golang.org/x/sys/unix"
	"os"
)

func reflink(src, dst string, perm os.FileMode) error {
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, perm)
	if err != nil {
		return err
	}
	err = unix.IoctlFileClone(int(out.Fd()), int(in.Fd()))
	closeErr := out.Close()
	if err != nil {
		_ = os.Remove(dst)
		return err
	}
	if closeErr != nil {
		_ = os.Remove(dst)
		return closeErr
	}
	return nil
}
