package remap

import (
	"runtime"
	"strings"
	"testing"
)

func TestWrapCommandIdempotent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no remap launcher on windows")
	}
	cmd := WrapCommand("httpserver 18082")
	if !strings.HasPrefix(cmd, LaunchPath()+" -c ") {
		t.Fatalf("wrap: %s", cmd)
	}
	if strings.Contains(cmd, "/bin/sh") {
		t.Fatal("launch must exec the target, not SIP-restricted /bin/sh")
	}
	if again := WrapCommand(cmd); again != cmd {
		t.Fatalf("not idempotent: %s vs %s", cmd, again)
	}
}

func TestWrapProcessCommands(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("no remap launcher on windows")
	}
	procs := map[string]any{
		"api": map[string]any{
			"command": "httpserver 8080",
			"readiness_probe": map[string]any{
				"exec": map[string]any{"command": "curl -sf http://127.0.0.1:8080/"},
			},
		},
	}
	WrapProcessCommands(procs)
	api := procs["api"].(map[string]any)
	if !strings.HasPrefix(api["command"].(string), LaunchPath()) {
		t.Fatalf("command: %v", api["command"])
	}
	probe := api["readiness_probe"].(map[string]any)["exec"].(map[string]any)["command"].(string)
	if !strings.HasPrefix(probe, LaunchPath()) {
		t.Fatalf("probe: %s", probe)
	}
}
