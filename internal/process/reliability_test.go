package process

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

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
	if err := Render(dir, map[string]any{"x": map[string]any{"command": "1234", "environment": []any{"VALUE=0001"}}}, nil); err != nil {
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
