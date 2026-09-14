package process

import (
	"context"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/Mrjwj34/berth/internal/config"
)

func TestRenderExpandsAndInjectsEnv(t *testing.T) {
	dir := t.TempDir()
	procs := map[string]any{
		"web": map[string]any{
			"command": "python3 -m http.server ${BERTH_PORT_WEB}",
			"readiness_probe": map[string]any{
				"http_get": map[string]any{
					"port": "${BERTH_PORT_WEB}",
					"path": "/",
				},
			},
		},
	}
	env := map[string]string{"BERTH_PORT_WEB": "20123", "BERTH_WORKSPACE": dir}
	if err := RenderAt(dir, dir, procs, env, 0); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(PCFile(dir))
	if err != nil {
		t.Fatal(err)
	}
	s := string(data)
	if !strings.Contains(s, "20123") || strings.Contains(s, "${BERTH_PORT_WEB}") {
		t.Fatalf("expansion failed:\n%s", s)
	}
	if !strings.Contains(s, "BERTH_PORT_WEB=20123") {
		t.Fatalf("env not injected:\n%s", s)
	}
}

func TestUpDownSimpleHTTP(t *testing.T) {
	t.Setenv("BERTH_HOME", t.TempDir())
	python := "python3"
	if runtime.GOOS == "windows" {
		python = "python"
	}
	if _, err := exec.LookPath(python); err != nil {
		if os.Getenv("BERTH_TEST_REQUIRE_NATIVE") == "1" {
			t.Fatal(err)
		}
		t.Skip(err)
	}
	if _, err := Ensure(context.Background()); err != nil {
		if os.Getenv("BERTH_TEST_REQUIRE_NATIVE") == "1" {
			t.Fatal(err)
		}
		t.Skip(err)
	}
	dir := t.TempDir()
	if err := os.MkdirAll(config.BerthDir(dir), 0o755); err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()
	env := map[string]string{
		"BERTH_PORT_WEB":  strconv.Itoa(port),
		"BERTH_WORKSPACE": dir,
		"BERTH_DATA_DIR":  config.DataDir(dir),
	}
	procs := map[string]any{
		"web": map[string]any{
			"command": python + " -m http.server ${BERTH_PORT_WEB} --bind 127.0.0.1",
		},
	}
	if err := RenderAt(dir, dir, procs, env, 0); err != nil {
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
	if running, err := IsRunning(ctx, dir); err != nil || !running {
		t.Fatalf("expected running, got %v (err: %v)", running, err)
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

// A process that ignores SIGTERM must not make down/done hang: process-compose
// has to escalate to SIGKILL after the configured shutdown timeout.
func TestDownEscalatesWhenProcessIgnoresSigterm(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("process groups and signals differ on Windows")
	}
	t.Setenv("BERTH_HOME", t.TempDir())
	python := "python3"
	if _, err := exec.LookPath(python); err != nil {
		if os.Getenv("BERTH_TEST_REQUIRE_NATIVE") == "1" {
			t.Fatal(err)
		}
		t.Skip(err)
	}
	if _, err := Ensure(context.Background()); err != nil {
		if os.Getenv("BERTH_TEST_REQUIRE_NATIVE") == "1" {
			t.Fatal(err)
		}
		t.Skip(err)
	}
	dir := t.TempDir()
	if err := os.MkdirAll(config.BerthDir(dir), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, config.Filename), []byte("version: 1\nshutdown_timeout_seconds: 2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	env := map[string]string{"BERTH_WORKSPACE": dir, "BERTH_DATA_DIR": config.DataDir(dir)}
	procs := map[string]any{
		"stubborn": map[string]any{
			"command": python + " -c 'import signal,time; signal.signal(signal.SIGTERM, signal.SIG_IGN); time.sleep(300)'",
		},
	}
	if err := RenderAt(dir, dir, procs, env, 2); err != nil {
		t.Fatal(err)
	}
	ctx := context.Background()
	if err := Up(ctx, dir, env, 0); err != nil {
		if data, readErr := os.ReadFile(LogFile(dir)); readErr == nil {
			t.Logf("supervisor log:\n%s", data)
		}
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = Down(context.Background(), dir) })
	start := time.Now()
	if err := Down(ctx, dir); err != nil {
		t.Fatalf("down did not escalate for a SIGTERM-ignoring process after %s: %v", time.Since(start), err)
	}
	if elapsed := time.Since(start); elapsed > 25*time.Second {
		t.Fatalf("down took %s; the shutdown timeout should bound it", elapsed)
	}
	if running, err := IsRunning(ctx, dir); err != nil || running {
		t.Fatalf("supervisor still running after down: %v, %v", running, err)
	}
}
