//go:build windows

package netns

import "os"

func processAlive(proc *os.Process) bool {
	return proc != nil
}
