package process

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestClientArgumentsNeverUseDefaultEndpoint(t *testing.T) {
	t.Setenv("BERTH_HOME", t.TempDir())
	dir := t.TempDir()
	if _, err := clientArgs(dir, 12345); err == nil {
		t.Fatal("missing token silently disables authentication")
	}
	if err := os.MkdirAll(filepath.Dir(TokenFile(dir)), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(TokenFile(dir), nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ensureToken(dir); err == nil {
		t.Fatal("empty token left by an interrupted write accepted")
	}
	if _, err := clientArgs(dir, 12345); err == nil {
		t.Fatal("empty token accepted by a control command")
	}
	if err := os.Remove(TokenFile(dir)); err != nil {
		t.Fatal(err)
	}
	if err := ensureToken(dir); err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS == "windows" {
		if _, err := clientArgs(dir, 0); err == nil {
			t.Fatal("missing control port falls back to an unrelated default endpoint")
		}
	}
	if _, err := clientArgs(dir, 12345); err != nil {
		t.Fatal(err)
	}
}

func TestControlFilesAreWorkspaceScoped(t *testing.T) {
	for _, suffix := range []string{"short", strings.Repeat("long", 30)} {
		t.Setenv("BERTH_HOME", filepath.Join(t.TempDir(), suffix))
		root := t.TempDir()
		a, b := filepath.Join(root, "alpha"), filepath.Join(root, "beta")
		if Socket(a) == Socket(b) || TokenFile(a) == TokenFile(b) {
			t.Fatal("parallel workspaces share a control endpoint or authentication file")
		}
	}
}

func TestUnknownAndFailedAreNotReady(t *testing.T) {
	for _, raw := range []string{`[{"name":"web","status":"Running","is_ready":"Unknown"}]`, `[{"name":"web","status":"Completed","exit_code":1,"is_ready":"N/A"}]`, `[]`} {
		p, err := DecodeStatus([]byte(raw))
		if err != nil {
			t.Fatal(err)
		}
		if allReady(p) {
			t.Fatalf("false readiness: %s", raw)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	if err := WaitReady(ctx, func(context.Context) ([]Proc, error) { return nil, nil }); err == nil {
		t.Fatal("empty process list treated as ready")
	}
}
func TestReadinessMatchesSupervisorProbeState(t *testing.T) {
	for _, tc := range []struct {
		name, status, ready string
		probe, want         bool
	}{
		{"running without probe", "Running", "-", false, true},
		{"probe pending", "Running", "-", true, false},
		{"probe ready", "Running", "Ready", true, true},
		{"probe failed", "Running", "Not Ready", true, false},
		{"unknown health", "Running", "Unknown", false, false},
		{"missing health", "Running", "", true, false},
		{"not running", "Pending", "-", false, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			raw := fmt.Sprintf(`[{"name":"web","status":%q,"is_ready":%q,"has_ready_probe":%t}]`, tc.status, tc.ready, tc.probe)
			procs, err := DecodeStatus([]byte(raw))
			if err != nil {
				t.Fatal(err)
			}
			if got := allReady(procs); got != tc.want {
				t.Fatalf("readiness = %t, want %t: %s", got, tc.want, raw)
			}
		})
	}
}
func TestRenderPreservesNumericStrings(t *testing.T) {
	dir := t.TempDir()
	if err := RenderAt(dir, dir, map[string]any{"x": map[string]any{"command": "1234", "environment": []any{"VALUE=0001"}}}, nil); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(PCFile(dir))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `command: "1234"`) {
		t.Fatalf("numeric string changed type: %s", data)
	}
}
func TestChecksumRejectsCorruption(t *testing.T) {
	p := filepath.Join(t.TempDir(), "binary")
	if err := os.WriteFile(p, []byte("bad"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := verifySHA256(p, strings.Repeat("0", 64)); err == nil {
		t.Fatal("accepted corrupt binary")
	}
}
