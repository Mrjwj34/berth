package remap

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestSeedAndLookup(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "remap.txt")
	if err := seedTable(path, []Mapping{{Name: "api", Listen: 8080, Host: 20100}}, []int{20101}); err != nil {
		t.Fatal(err)
	}
	host, ok := lookupHost(path, 8080)
	if !ok || host != 20100 {
		t.Fatalf("lookup: %d %v", host, ok)
	}
	if listen, ok := reverseListen(path, 20100); !ok || listen != 8080 {
		t.Fatalf("reverse: %d %v", listen, ok)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "#reserved 20101") {
		t.Fatalf("reserved missing:\n%s", data)
	}
}

func TestRewriteProbes(t *testing.T) {
	procs := map[string]any{
		"api": map[string]any{
			"command": "go run .",
			"readiness_probe": map[string]any{
				"http_get": map[string]any{"port": 8080, "path": "/healthz"},
			},
		},
	}
	out := RewriteProbes(procs, []Mapping{{Listen: 8080, Host: 20100}})
	probe := out["api"].(map[string]any)["readiness_probe"].(map[string]any)["http_get"].(map[string]any)
	if probe["port"] != 20100 {
		t.Fatalf("probe port = %v", probe["port"])
	}
	orig := procs["api"].(map[string]any)["readiness_probe"].(map[string]any)["http_get"].(map[string]any)
	if orig["port"] != 8080 {
		t.Fatal("rewrote the caller's process map")
	}
}

func TestPreloadBindAndPublicConnect(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("preload remap is linux/darwin")
	}
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available")
	}
	home := t.TempDir()
	t.Setenv("LANE_HOME", home)
	ctx := context.Background()
	lib, err := EnsureLib(ctx)
	if err != nil {
		t.Fatal(err)
	}
	dir := t.TempDir()
	host := freePort(t)
	listen := 18090
	if !portFree(listen) {
		t.Skip("18090 in use")
	}
	if err := Setup(ctx, dir, []Mapping{{Name: "api", Listen: listen, Host: host}}, nil); err != nil {
		t.Fatal(err)
	}
	env := append(os.Environ(),
		"LANE_REMAP=1",
		"LANE_REMAP_FILE="+TablePath(dir),
	)
	if runtime.GOOS == "darwin" {
		env = append(env, "DYLD_INSERT_LIBRARIES="+lib)
	} else {
		env = append(env, "LD_PRELOAD="+lib)
	}
	cmd := exec.Command("python3", "-m", "http.server", strconv.Itoa(listen), "--bind", "127.0.0.1")
	cmd.Env = env
	cmd.Dir = dir
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = cmd.Process.Kill(); _ = cmd.Wait() }()
	if !httpOK(t, host) {
		t.Fatalf("published %d did not serve", host)
	}
	if !portFree(listen) {
		t.Fatal("hardcoded listen port leaked onto the host")
	}

	pub := exec.Command("python3", "-c", `
import socket, sys
s = socket.socket()
s.settimeout(5)
s.connect(("1.1.1.1", 443))
s.close()
print("ok")
`)
	pub.Env = env
	out, err := pub.CombinedOutput()
	if err != nil {
		t.Skipf("public connect skipped: %v\n%s", err, out)
	}
	if !strings.Contains(string(out), "ok") {
		t.Fatalf("public connect: %s", out)
	}
}

func TestLinuxGoBindRemap(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("seccomp wrap is linux-only")
	}
	home := t.TempDir()
	t.Setenv("LANE_HOME", home)
	ctx := context.Background()
	if _, err := EnsureLib(ctx); err != nil {
		t.Fatal(err)
	}
	helper := filepath.Join(t.TempDir(), "hardlisten")
	src := filepath.Join("testdata", "hardlisten", "main.go")
	build := exec.Command("go", "build", "-o", helper, src)
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build helper: %v\n%s", err, out)
	}
	dir := t.TempDir()
	host := freePort(t)
	listen := 18093
	if !portFree(listen) {
		t.Skip("18093 in use")
	}
	if err := Setup(ctx, dir, []Mapping{{Name: "api", Listen: listen, Host: host}}, nil); err != nil {
		t.Fatal(err)
	}
	t.Setenv("LANE_REMAP_FILE", TablePath(dir))
	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()
	done := make(chan error, 1)
	go func() {
		done <- Supervise(runCtx, []string{helper, strconv.Itoa(listen)})
	}()
	if !httpOK(t, host) {
		t.Fatalf("go listen %d was not published on %d", listen, host)
	}
	if !portFree(listen) {
		t.Fatal("hardcoded listen port leaked onto the host")
	}
	cancel()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
	}
}

func TestTwoWorkspacesSameListenPort(t *testing.T) {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		t.Skip("preload remap is linux/darwin")
	}
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available")
	}
	home := t.TempDir()
	t.Setenv("LANE_HOME", home)
	ctx := context.Background()
	if _, err := EnsureLib(ctx); err != nil {
		t.Fatal(err)
	}
	listen := 18091
	if !portFree(listen) {
		t.Skip("18091 in use")
	}
	a := startRemappedHTTP(t, ctx, listen)
	defer a.stop()
	b := startRemappedHTTP(t, ctx, listen)
	defer b.stop()
	if !httpOK(t, a.host) || !httpOK(t, b.host) {
		t.Fatal("parallel remapped servers failed")
	}
	if !portFree(listen) {
		t.Fatal("hardcoded listen port leaked onto the host")
	}
}

type remappedHTTP struct {
	cmd  *exec.Cmd
	host int
}

func (r remappedHTTP) stop() {
	if r.cmd != nil && r.cmd.Process != nil {
		_ = r.cmd.Process.Kill()
		_ = r.cmd.Wait()
	}
}

func startRemappedHTTP(t *testing.T, ctx context.Context, listen int) remappedHTTP {
	t.Helper()
	dir := t.TempDir()
	host := freePort(t)
	if err := Setup(ctx, dir, []Mapping{{Name: "api", Listen: listen, Host: host}}, nil); err != nil {
		t.Fatal(err)
	}
	env := ApplyMap(map[string]string{}, dir)
	cmd := exec.Command("python3", "-m", "http.server", strconv.Itoa(listen), "--bind", "127.0.0.1")
	cmd.Dir = dir
	cmd.Env = Environ(os.Environ(), dir)
	_ = env
	if err := cmd.Start(); err != nil {
		t.Fatal(err)
	}
	if !httpOK(t, host) {
		_ = cmd.Process.Kill()
		t.Fatalf("server on host %d did not become ready", host)
	}
	return remappedHTTP{cmd: cmd, host: host}
}

func httpOK(t *testing.T, port int) bool {
	t.Helper()
	url := fmt.Sprintf("http://127.0.0.1:%d/", port)
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			_, _ = io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode == 200 {
				return true
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	return false
}

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	_ = ln.Close()
	return port
}

func portFree(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	_ = ln.Close()
	return true
}
