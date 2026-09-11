package remap

import (
	"os/exec"
	"runtime"
	"strings"
)

// WrapCommand prefixes a process-compose command with the unsigned launcher
// so DYLD_INSERT_LIBRARIES is set after a restricted parent (signed Go
// binaries, process-compose, /bin/sh) has already stripped DYLD_* .
// process-compose already runs the string via bash -c, so we must not add
// another -c/wordexp layer (macOS wordexp+WRDE_NOCMD returns 127).
func WrapCommand(command string) string {
	if runtime.GOOS == "windows" || command == "" {
		return command
	}
	launch := LaunchPath()
	if strings.HasPrefix(command, launch+" ") || command == launch {
		return command
	}
	return launch + " " + command
}

// WrapProcessCommands rewrites each process command in a pc.yaml document.
func WrapProcessCommands(procs map[string]any) {
	if runtime.GOOS == "windows" {
		return
	}
	for _, raw := range procs {
		proc, ok := raw.(map[string]any)
		if !ok {
			continue
		}
		if cmd, ok := proc["command"].(string); ok && cmd != "" {
			proc["command"] = WrapCommand(cmd)
		}
		wrapExecProbe(proc)
	}
}

func wrapExecProbe(proc map[string]any) {
	probe, ok := proc["readiness_probe"].(map[string]any)
	if !ok {
		return
	}
	ex, ok := probe["exec"].(map[string]any)
	if !ok {
		return
	}
	cmd, ok := ex["command"].(string)
	if !ok || cmd == "" {
		return
	}
	ex["command"] = WrapCommand(cmd)
}

// LaunchCmd runs name after the unsigned launcher so remap preload is applied.
func LaunchCmd(name string, args ...string) *exec.Cmd {
	argv := append([]string{name}, args...)
	return exec.Command(LaunchPath(), argv...)
}
