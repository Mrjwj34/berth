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
	"time"

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
	t.Setenv("LANE_HOME", t.TempDir())
	python := "python3"
	if runtime.GOOS == "windows" {
		python = "python"
	}
	if _, err := exec.LookPath(python); err != nil {
		if os.Getenv("LANE_TEST_REQUIRE_NATIVE") == "1" {
			t.Fatal(err)
		}
		t.Skip(err)
	}
	if _, err := Ensure(context.Background()); err != nil {
		if os.Getenv("LANE_TEST_REQUIRE_NATIVE") == "1" {
			t.Fatal(err)
		}
		t.Skip(err)
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
			"command": python + " -m http.server ${LANE_PORT_WEB} --bind 127.0.0.1",
		},
	}
	if err := Render(dir, procs, env); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	pcPort := 0
	if runtime.GOOS == "windows" {
		l, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			t.Fatal(err)
		}
		pcPort = l.Addr().(*net.TCPAddr).Port
		_ = l.Close()
	}
	t.Cleanup(func() {
		if err := Down(context.Background(), dir); err != nil {
			t.Errorf("stop test supervisor: %v", err)
		}
	})
	if err := Up(ctx, dir, env, pcPort); err != nil {
		if data, readErr := os.ReadFile(LogFile(dir)); readErr == nil {
			t.Logf("supervisor log:\n%s", data)
		}
		if data, readErr := os.ReadFile(PCFile(dir)); readErr == nil {
			t.Logf("generated configuration:\n%s", data)
		}
		bin, _ := LookPath()
		debugCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		control, controlErr := clientArgs(dir, readPCPort(dir))
		if controlErr != nil {
			t.Fatalf("startup: %v; control: %v", err, controlErr)
		}
		args := append([]string{"process", "list", "-o", "json"}, control...)
		debug := exec.CommandContext(debugCtx, bin, args...)
		debug.Env = pcEnviron(os.Environ())
		out, debugErr := debug.CombinedOutput()
		t.Logf("control endpoint=%s; query=%v; output=%s", Socket(dir), debugErr, out)
		t.Fatal(err)
	}
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
