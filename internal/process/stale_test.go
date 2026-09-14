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

	"github.com/Mrjwj34/berth/internal/config"
)

func TestIsRunningReclaimsSupervisorThatExitedOnItsOwn(t *testing.T) {
	dir := departedSupervisor(t)
	running, err := IsRunning(context.Background(), dir)
	if err != nil || running {
		t.Fatalf("a supervisor that exited on its own must count as stopped: running=%v err=%v", running, err)
	}
	assertReclaimed(t, dir)
}

func TestIsRunningReclaimsSupervisorWithDeadRecordedPID(t *testing.T) {
	dir := departedSupervisor(t)
	if err := writePCPID(dir, deadPID(t)); err != nil {
		t.Fatal(err)
	}
	running, err := IsRunning(context.Background(), dir)
	if err != nil || running {
		t.Fatalf("a dead recorded supervisor must count as stopped: running=%v err=%v", running, err)
	}
	assertReclaimed(t, dir)
}

func TestIsRunningKeepsUnknownStateWhileTheEndpointAnswers(t *testing.T) {
	dir := workspace(t)
	serveControl(t, dir)
	running, err := IsRunning(context.Background(), dir)
	if running || err == nil || !strings.Contains(err.Error(), "process state unknown") {
		t.Fatalf("an endpoint that answers must stay unknown: running=%v err=%v", running, err)
	}
	if _, err := os.Lstat(controlArtifact(dir)); err != nil {
		t.Fatalf("control files must survive an unresponsive supervisor: %v", err)
	}
}

func TestIsRunningKeepsUnknownStateWhileRecordedSupervisorIsAlive(t *testing.T) {
	dir := departedSupervisor(t)
	// The window this guards: a supervisor berth just spawned may not be
	// listening yet, so a refused probe alone must not reclaim its files.
	if err := writePCPID(dir, os.Getpid()); err != nil {
		t.Fatal(err)
	}
	running, err := IsRunning(context.Background(), dir)
	if running || err == nil || !strings.Contains(err.Error(), "process state unknown") {
		t.Fatalf("a live recorded supervisor must stay unknown: running=%v err=%v", running, err)
	}
	if _, err := os.Lstat(controlArtifact(dir)); err != nil {
		t.Fatalf("control files must survive a live recorded supervisor: %v", err)
	}
}

func TestIsRunningKeepsUnknownStateWithoutUsableEndpoint(t *testing.T) {
	dir := workspace(t)
	if runtime.GOOS == "windows" {
		// A torn or foreign port file names no address to probe.
		if err := os.WriteFile(PortFile(dir), []byte("stale endpoint, not a live supervisor"), 0o600); err != nil {
			t.Fatal(err)
		}
	} else if err := os.WriteFile(Socket(dir), []byte("not a socket"), 0o600); err != nil {
		t.Fatal(err)
	}
	running, err := IsRunning(context.Background(), dir)
	if running || err == nil || !strings.Contains(err.Error(), "process state unknown") {
		t.Fatalf("an unaddressable endpoint proves nothing: running=%v err=%v", running, err)
	}
	if _, err := os.Lstat(controlArtifact(dir)); err != nil {
		t.Fatalf("control files must survive an unaddressable endpoint: %v", err)
	}
}

// workspace isolates BERTH_HOME so no test touches the machine registry or home.
func workspace(t *testing.T) string {
	t.Helper()
	t.Setenv("BERTH_HOME", t.TempDir())
	dir := t.TempDir()
	if err := os.MkdirAll(config.BerthDir(dir), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(Socket(dir)), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := ensureToken(dir); err != nil {
		t.Fatal(err)
	}
	return dir
}

// departedSupervisor leaves behind exactly what a supervisor that completed or
// crashed leaves: control files that nothing serves.
func departedSupervisor(t *testing.T) string {
	t.Helper()
	dir := workspace(t)
	if runtime.GOOS == "windows" {
		if err := writePort(t, dir, freePort(t)); err != nil {
			t.Fatal(err)
		}
		return dir
	}
	staleSocket(t, dir)
	return dir
}

func staleSocket(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(Socket(dir)), 0o700); err != nil {
		t.Fatal(err)
	}
	ln, err := net.Listen("unix", Socket(dir))
	if err != nil {
		t.Fatal(err)
	}
	ln.(*net.UnixListener).SetUnlinkOnClose(false)
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}
}

// serveControl binds the control endpoint and answers nothing, which is the
// unresponsive supervisor the unknown-state guard exists for.
func serveControl(t *testing.T, dir string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(Socket(dir)), 0o700); err != nil {
		t.Fatal(err)
	}
	var ln net.Listener
	var err error
	if runtime.GOOS == "windows" {
		port := freePort(t)
		ln, err = net.Listen("tcp", net.JoinHostPort("127.0.0.1", strconv.Itoa(port)))
		if err != nil {
			t.Fatal(err)
		}
		if err := writePort(t, dir, port); err != nil {
			t.Fatal(err)
		}
	} else if ln, err = net.Listen("unix", Socket(dir)); err != nil {
		t.Fatal(err)
	}
	go func() {
		for {
			conn, err := ln.Accept()
			if err != nil {
				return
			}
			_ = conn.Close()
		}
	}()
	t.Cleanup(func() { _ = ln.Close() })
}

func writePort(t *testing.T, dir string, port int) error {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(PortFile(dir)), 0o700); err != nil {
		return err
	}
	return os.WriteFile(PortFile(dir), []byte(strconv.Itoa(port)), 0o600)
}

func freePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatal(err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	if err := ln.Close(); err != nil {
		t.Fatal(err)
	}
	return port
}

// deadPID returns a process identifier that is certainly no longer running.
func deadPID(t *testing.T) int {
	t.Helper()
	name, args := "cmd", []string{"/c", "exit"}
	if runtime.GOOS != "windows" {
		// /bin/true does not exist on macOS, and `true` is a shell builtin
		// there, so run the shell itself.
		name, args = "sh", []string{"-c", "exit 0"}
	}
	cmd := exec.Command(name, args...)
	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
	return cmd.Process.Pid
}

func controlArtifact(dir string) string {
	if runtime.GOOS == "windows" {
		return PortFile(dir)
	}
	return Socket(dir)
}

func assertReclaimed(t *testing.T, dir string) {
	t.Helper()
	for _, path := range []string{Socket(dir), PortFile(dir), PIDFile(dir)} {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatalf("stale control file %s not reclaimed (err=%v)", path, err)
		}
	}
}
