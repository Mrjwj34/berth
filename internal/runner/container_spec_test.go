package runner

import (
	"testing"

	"github.com/Mrjwj34/berth/internal/config"
	"github.com/Mrjwj34/berth/internal/state"
)

// A workspace that declares no ports is persisted without its port maps
// (omitempty) and therefore reads back with nil maps, while the record used when
// its container was created held empty ones. Both shapes must hash identically:
// otherwise the container fails the ownership check in inspect() and berth
// refuses to control a runtime it created itself, which is exactly what happened
// to a project whose berth.yaml declares neither ports nor processes.
func TestSpecHashTreatsEmptyAndAbsentMapsAlike(t *testing.T) {
	base := state.Workspace{
		ID:      "abc",
		Path:    "/tmp/project.berths/one",
		GitDir:  "/tmp/project/.git/worktrees/one",
		Runtime: config.Runtime{Backend: "container", Engine: "docker", Image: "berth-runtime:test"},
	}
	absent := base
	empty := base
	empty.Ports = map[string]int{}
	empty.Listen = map[string]int{}

	absentHash := (&Session{Workspace: absent}).specHash()
	if (&Session{Workspace: empty}).specHash() != absentHash {
		t.Fatal("empty and absent port maps must hash the same")
	}

	withPorts := base
	withPorts.Ports = map[string]int{"web": 20000}
	withListen := base
	withListen.Listen = map[string]int{"web": 8080}
	for name, workspace := range map[string]state.Workspace{"ports": withPorts, "listen": withListen} {
		if (&Session{Workspace: workspace}).specHash() == absentHash {
			t.Fatalf("declared %s must change the hash", name)
		}
	}
}
