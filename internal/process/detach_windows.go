//go:build windows

package process

import (
	"os/exec"
	"syscall"

	"golang.org/x/sys/windows"
)

// configureDetached starts the supervisor so that it, and everything it spawns,
// stays off screen.
//
// CREATE_NO_WINDOW gives the supervisor a console that Windows never displays.
// process-compose starts every process as `cmd /C <line>`, and a console
// application launched without a console makes Windows allocate a visible one,
// which is why this used to pop a console window for every process of every
// running workspace. Children of the supervisor now attach to its hidden
// console instead.
//
// The two flags are mutually exclusive: Windows ignores CREATE_NO_WINDOW when
// DETACHED_PROCESS is also set, so this replaces the detached flag rather than
// adding to it. Survival across terminal close is unchanged, because the hidden
// console belongs to the supervisor rather than to the terminal berth ran in.
// CREATE_NEW_PROCESS_GROUP keeps a Ctrl+C in that terminal away from the
// supervisor and its children.
func configureDetached(cmd *exec.Cmd) {
	cmd.SysProcAttr = &syscall.SysProcAttr{
		CreationFlags: windows.CREATE_NEW_PROCESS_GROUP | windows.CREATE_NO_WINDOW,
	}
}
