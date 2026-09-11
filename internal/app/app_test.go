package app

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Mrjwj34/lane/internal/gitx"
	"github.com/Mrjwj34/lane/internal/state"
)

func TestNewLSDoneLifecycle(t *testing.T) {
	home := t.TempDir()
	t.Setenv("LANE_HOME", home)
	repo := initRepo(t)
	if err := os.Chdir(repo); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir("/") })

	ctx := context.Background()
	a, err := Open(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()

	ws, err := a.New(ctx, "feat-x", "main", false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(ws.Path); err != nil {
		t.Fatal(err)
	}
	if ws.Branch != "lane/feat-x" {
		t.Fatalf("branch = %s", ws.Branch)
	}

	again, err := a.New(ctx, "feat-x", "main", false)
	if err != nil {
		t.Fatal(err)
	}
	if again.Path != ws.Path {
		t.Fatalf("new is not idempotent: %s vs %s", again.Path, ws.Path)
	}

	list, err := a.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 1 {
		t.Fatalf("list = %d", len(list))
	}

	if err := os.Chdir(ws.Path); err != nil {
		t.Fatal(err)
	}
	st, err := a.Status(ctx, "")
	if err != nil {
		t.Fatal(err)
	}
	if st.Slug != "feat-x" {
		t.Fatalf("status slug = %s", st.Slug)
	}

	if err := a.Done(ctx, "feat-x", true); err != nil {
		t.Fatal(err)
	}
	list, err = a.List(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(list) != 0 {
		t.Fatalf("expected empty list after done, got %d", len(list))
	}
	file, err := a.Store.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := file.Lookup(ws.Path); ok {
		t.Fatal("state still has workspace")
	}
}

func TestGCReclaimsVanished(t *testing.T) {
	t.Setenv("LANE_HOME", t.TempDir())
	ctx := context.Background()
	a, err := Open(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()
	ghost := filepath.Join(t.TempDir(), "missing-ws")
	if err := a.Store.Update(ctx, func(f *state.File) error {
		f.Workspaces[ghost] = state.Workspace{Slug: "ghost", Path: ghost, Repo: t.TempDir()}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	rep, err := a.GC(ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Actions) == 0 {
		t.Fatal("expected reclaim action")
	}
	file, err := a.Store.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := file.Workspaces[ghost]; ok {
		t.Fatal("ghost workspace remains")
	}
}

func initRepo(t *testing.T) string {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		if _, err := gitx.Run(context.Background(), dir, args...); err != nil {
			t.Fatal(err)
		}
	}
	run("init", "-b", "main")
	run("config", "user.email", "lane@test")
	run("config", "user.name", "lane")
	if err := os.WriteFile(filepath.Join(dir, "README"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "lane.yaml"), []byte("version: 1\nbase: main\nports: [web]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", ".")
	run("commit", "-m", "init")
	return dir
}
