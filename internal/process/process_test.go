package process

import (
	"context"
	"net"
	"os"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
	"testing"

	"github.com/Mrjwj34/lane/internal/config"
)

func TestRenderExpandsAndInjectsEnv(t *testing.T) {
	dir := t.TempDir()
	procs := map[string]any{
		"web": map[string]any{
			"command": "python3 -m http.server ${LANE_PORT_WEB}",
			"readiness_probe": map[string]any{
				"http_get": map[string]any{
					"port": "${LANE_PORT_WEB}",
					"path": "/",
				},
			},
		},
	}
	env := map[string]string{"LANE_PORT_WEB": "20123", "LANE_WORKSPACE": dir}
	if err := Render(dir, procs, env); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(PCFile(dir))
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, "20123") || strings.Contains(s, "${LANE_PORT_WEB}") {
		t.Fatalf("expansion failed:\n%s", s)
	}
	if !strings.Contains(s, "LANE_PORT_WEB=20123") {
		t.Fatalf("env not injected:\n%s", s)
	}
}

func TestUpDownSimpleHTTP(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("unix socket path in this test")
	}
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available")
	}
	if _, err := LookPath(); err != nil {
		if _, err := Ensure(context.Background()); err != nil {
			t.Skip(err)
		}
	}
	dir := t.TempDir()
	if err := os.MkdirAll(config.LaneDir(dir), 0o755); err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()
	env := map[string]string{
		"LANE_PORT_WEB":  strconv.Itoa(port),
		"LANE_WORKSPACE": dir,
		"LANE_DATA_DIR":  config.DataDir(dir),
	}
	procs := map[string]any{
		"web": map[string]any{
			"command": "python3 -m http.server ${LANE_PORT_WEB} --bind 127.0.0.1",
		},
	}
	if err := Render(dir, procs, env); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := Up(ctx, dir, env, 0, nil); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = Down(ctx, dir) }()
	if !Running(ctx, dir) {
		t.Fatal("expected running")
	}
	st, err := Status(ctx, dir)
	if err != nil {
		t.Fatal(err)
	}
	if len(st) == 0 {
		t.Fatal("no processes in status")
	}
	if err := Down(ctx, dir); err != nil {
		t.Fatal(err)
	}
}
