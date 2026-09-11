package runner

import (
	"context"
	"github.com/Mrjwj34/berth/internal/config"
	"github.com/Mrjwj34/berth/internal/state"
	"path/filepath"
	"strings"
	"testing"
)

func fixture(t *testing.T) *Session {
	t.Helper()
	root := t.TempDir()
	r := config.Runtime{Backend: "container", Image: "berth-test:local", Memory: "1g", CPUs: 2}
	ws := state.Workspace{ID: "0123456789abcdef", Slug: "test", Path: filepath.Join(root, "worktree"), GitDir: filepath.Join(root, "repo", ".git", "worktrees", "test"), Ports: map[string]int{"web": 20100}, Listen: map[string]int{"web": 30000}, Runtime: r}
	s, err := New(&config.Config{Ports: []string{"web"}, Listen: ws.Listen, Runtime: r}, ws)
	if err != nil {
		t.Fatal(err)
	}
	return s
}
func TestCreateArgumentsAndEnv(t *testing.T) {
	s := fixture(t)
	args, err := s.createArgs()
	if err != nil {
		t.Fatal(err)
	}
	joined := strings.Join(args, "\n")
	for _, want := range []string{"--pull=never", "--init", "127.0.0.1:20100:65535/tcp", "TCP:127.0.0.1:30000", "--memory\n1g", "--cpus\n2", "wait -n"} {
		if !strings.Contains(joined, want) {
			t.Fatalf("missing %q in %v", want, args)
		}
	}
	if strings.Contains(joined, "--privileged") || strings.Contains(joined, "docker.sock") {
		t.Fatal("excessive runtime privileges")
	}
	env, host := s.Env(), s.HostEnv()
	if env["BERTH_PORT_WEB"] != "30000" || env["BERTH_HOST_PORT_WEB"] != "20100" || host["BERTH_PORT_WEB"] != "20100" {
		t.Fatal("host and internal ports conflated")
	}
	if env["BERTH_DATA_DIR"] != "/workspace/.berth/data" || !strings.HasPrefix(env["GIT_DIR"], "/berth/git/worktrees/") {
		t.Fatalf("host path leaked into container: %+v", env)
	}
	cmd := s.containerExec(context.Background(), []string{"sh", "-c", "echo '$HOME' && echo end"}, false)
	if cmd.Args[len(cmd.Args)-1] != "echo '$HOME' && echo end" {
		t.Fatal("shell command reparsed")
	}
	if !s.Plan().NetworkNamespace || s.Plan().SecuritySandbox {
		t.Fatal("incorrect guarantees")
	}
}
func TestRuntimeChangesRejected(t *testing.T) {
	s := fixture(t)
	cfg := *s.Config
	cfg.Runtime.Image = "different"
	if _, err := New(&cfg, s.Workspace); err == nil {
		t.Fatal("runtime contract changed silently")
	}
}
func TestMissingEngineFailsClosed(t *testing.T) {
	s := fixture(t)
	s.Workspace.Runtime.Engine = filepath.Join(t.TempDir(), "missing-engine")
	if err := s.Prepare(context.Background()); err == nil {
		t.Fatal("fell back to host")
	}
}
