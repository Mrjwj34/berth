//go:build unix

package netns

import (
	"os"
	"syscall"
)

func processAlive(proc *os.Process) bool {
	return proc.Signal(syscall.Signal(0)) == nil
}
