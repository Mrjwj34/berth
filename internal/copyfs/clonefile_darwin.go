//go:build darwin

package copyfs

import "golang.org/x/sys/unix"

func clonefile(src, dst string) error {
	return unix.Clonefile(src, dst, 0)
}
