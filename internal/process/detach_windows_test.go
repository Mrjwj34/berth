//go:build windows

package process

import (
	"os/exec"
	"testing"

	"golang.org/x/sys/windows"
)

// A detached supervisor has no console to inherit, so Windows hands every
// `cmd /C` child its own visible console window. The supervisor therefore has
// to own a hidden console instead, and the flags that would undo that are
// asserted against explicitly.
func TestConfigureDetachedKeepsChildrenOffScreen(t *testing.T) {
	cmd := exec.Command("cmd", "/c", "exit 0")
	configureDetached(cmd)

	if cmd.SysProcAttr == nil {
		t.Fatal("configureDetached left SysProcAttr unset, so children get their own consoles")
	}
	flags := cmd.SysProcAttr.CreationFlags
	if flags&windows.CREATE_NO_WINDOW == 0 {
		t.Fatalf("CreationFlags %#x does not include CREATE_NO_WINDOW", flags)
	}
	if flags&windows.DETACHED_PROCESS != 0 {
		t.Fatalf("CreationFlags %#x includes DETACHED_PROCESS, which cancels CREATE_NO_WINDOW", flags)
	}
	if flags&windows.CREATE_NEW_PROCESS_GROUP == 0 {
		t.Fatalf("CreationFlags %#x does not include CREATE_NEW_PROCESS_GROUP", flags)
	}
}
