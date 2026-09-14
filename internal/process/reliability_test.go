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

	"github.com/Mrjwj34/berth/internal/config"
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
	if err := RenderAt(dir, dir, map[string]any{"x": map[string]any{"command": "1234", "environment": []any{"VALUE=0001"}}}, nil, 0); err != nil {
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

func TestRenderBoundsShutdownAndKeepsProjectBlock(t *testing.T) {
	dir := t.TempDir()
	procs := map[string]any{
		"plain": map[string]any{"command": "sleep 1"},
		"custom": map[string]any{
			"command":  "sleep 1",
			"shutdown": map[string]any{"signal": 9},
		},
	}
	if err := RenderAt(dir, dir, procs, nil, 7); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(PCFile(dir))
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	if !strings.Contains(text, "timeout_seconds: 7") {
		t.Fatalf("shutdown timeout not injected:\n%s", text)
	}
	if !strings.Contains(text, "signal: 9") {
		t.Fatalf("project shutdown block overwritten:\n%s", text)
	}
	if strings.Count(text, "timeout_seconds: 7") != 1 {
		t.Fatalf("injected shutdown bounds into a project-declared block:\n%s", text)
	}
}

func TestDownTimeoutHonoursConfig(t *testing.T) {
	dir := t.TempDir()
	if got := downTimeout(dir); got != 35*time.Second {
		t.Fatalf("default down timeout = %s, want 35s", got)
	}
	if err := os.WriteFile(filepath.Join(dir, config.Filename), []byte("version: 1\nshutdown_timeout_seconds: 90\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := downTimeout(dir); got != 110*time.Second {
		t.Fatalf("configured down timeout = %s, want 110s", got)
	}
	if err := os.WriteFile(filepath.Join(dir, config.Filename), []byte("version: 1\nshutdown_timeout_seconds: 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if got := downTimeout(dir); got != 30*time.Second {
		t.Fatalf("down timeout floor = %s, want 30s", got)
	}
}

func TestCleanSupervisorOutputDropsDebugNoise(t *testing.T) {
	raw := []byte("{\"level\":\"debug\",\"message\":\"Path not found\"}\nprocess-compose down: signal: killed\n")
	if got := cleanSupervisorOutput(raw); got != "process-compose down: signal: killed" {
		t.Fatalf("cleaned output = %q", got)
	}
}
