//go:build linux

package netns

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"testing"
	"time"
)

func TestAvailable(t *testing.T) {
	if !Available() {
		t.Skip("unprivileged user+net namespaces are disabled")
	}
}

func TestTwoIsolatesSameListenPort(t *testing.T) {
	if !Available() {
		t.Skip("unprivileged user+net namespaces are disabled")
	}
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available")
	}
	listen := 18080
	if !free(listen) {
		t.Skip("18080 is already bound on the host")
	}
	ctx := context.Background()
	a := startHTTP(t, ctx, listen)
	defer func() { _ = Stop(a.dir) }()
	b := startHTTP(t, ctx, listen)
	defer func() { _ = Stop(b.dir) }()

	if got := get(t, a.host); got == "" {
		t.Fatal("workspace a did not serve")
	}
	if got := get(t, b.host); got == "" {
		t.Fatal("workspace b did not serve")
	}
	if !free(listen) {
		t.Fatal("hardcoded listen port leaked onto the host")
	}
}

func TestIsolateDiscoversWithoutListenMaps(t *testing.T) {
	if !Available() {
		t.Skip("unprivileged user+net namespaces are disabled")
	}
	if _, err := exec.LookPath("python3"); err != nil {
		t.Skip("python3 not available")
	}
	listen := 18081
	if !free(listen) {
		t.Skip("18081 is already bound on the host")
	}
	ctx := context.Background()
	dir := t.TempDir()
	if err := Start(ctx, dir, nil, true, "python3", []string{"-m", "http.server", strconv.Itoa(listen), "--bind", "127.0.0.1"}, os.Environ()); err != nil {
		t.Fatal(err)
	}
	defer func() { _ = Stop(dir) }()
	deadline := time.Now().Add(8 * time.Second)
	var found []int
	var err error
	for time.Now().Before(deadline) {
		found, err = Discover(dir)
		if err == nil {
			for _, p := range found {
				if p == listen {
					host := freePort(t)
					if err := Publish(dir, Mapping{Name: strconv.Itoa(listen), Host: host, Listen: listen}); err != nil {
						t.Fatal(err)
					}
					if get(t, host) == "" {
						t.Fatal("discovered publish did not serve")
					}
					return
				}
			}
		}
		time.Sleep(80 * time.Millisecond)
	}
	t.Fatalf("did not discover listen %d (found %v err %v)", listen, found, err)
}

func startHTTP(t *testing.T, ctx context.Context, listen int) struct {
	dir  string
	host int
} {
	t.Helper()
	dir := t.TempDir()
	host := freePort(t)
	err := Start(ctx, dir, []Mapping{{Name: "api", Host: host, Listen: listen}}, true,
		"python3", []string{"-m", "http.server", strconv.Itoa(listen), "--bind", "127.0.0.1"}, os.Environ())
	if err != nil {
		t.Fatal(err)
	}
	return struct {
		dir  string
		host int
	}{dir: dir, host: host}
}

func get(t *testing.T, port int) string {
	t.Helper()
	url := fmt.Sprintf("http://127.0.0.1:%d/", port)
	deadline := time.Now().Add(8 * time.Second)
	var last error
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			body, _ := io.ReadAll(resp.Body)
			_ = resp.Body.Close()
			if resp.StatusCode == 200 {
				return string(body)
			}
			last = fmt.Errorf("status %d", resp.StatusCode)
		} else {
			last = err
		}
		time.Sleep(80 * time.Millisecond)
	}
	t.Fatalf("GET %s: %v", url, last)
	return ""
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

func free(port int) bool {
	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", port))
	if err != nil {
		return false
	}
	_ = ln.Close()
	return true
}
