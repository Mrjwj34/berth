//go:build !windows

package process

import "os/exec"

func configureDetached(_ *exec.Cmd) {}
