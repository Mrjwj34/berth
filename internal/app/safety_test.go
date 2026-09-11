package app

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/Mrjwj34/berth/internal/gitx"
	"github.com/Mrjwj34/berth/internal/process"
	"github.com/Mrjwj34/berth/internal/state"
)

func testApp(t *testing.T) (*App, string) {
	t.Helper()
	t.Setenv("BERTH_HOME", t.TempDir())
	repo := initRepo(t)
	old, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(repo); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(old) })
	a, err := Open(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = a.Close() })
	return a, repo
}
func writeConfig(t *testing.T, repo, text string) {
	t.Helper()
	if err := os.WriteFile(filepath.Join(repo, "berth.yaml"), []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "berth.yaml"}, {"commit", "-m", "configure runtime"}} {
		if _, err := gitx.Run(context.Background(), repo, args...); err != nil {
			t.Fatal(err)
		}
	}
}
func TestPrimaryWorktreeCannotBeAdoptedOrForcedAway(t *testing.T) {
	a, repo := testApp(t)
	ctx := context.Background()
	if _, err := a.Adopt(ctx, true); err == nil {
		t.Fatal("adopt accepted primary worktree")
	}
	ws := state.Workspace{ID: "1234567890abcdef", Slug: "primary", Path: mustKey(repo), Repo: mustKey(repo), Branch: "main", Ownership: state.Owned}
	if err := a.Store.Update(ctx, func(f *state.File) error { f.Workspaces[ws.Path] = ws; return nil }); err != nil {
		t.Fatal(err)
	}
	if err := a.Done(ctx, ws.Path, true); err == nil {
		t.Fatal("forced done accepted primary checkout")
	}
	if _, err := os.Stat(filepath.Join(repo, ".git")); err != nil {
		t.Fatalf("primary repository lost: %v", err)
	}
}
func TestAdoptedCheckoutIsPreservedEvenWithForce(t *testing.T) {
	a, repo := testApp(t)
	ctx := context.Background()
	path := filepath.Join(t.TempDir(), "external")
	if _, err := gitx.Run(ctx, repo, "worktree", "add", "-b", "external", path, "main"); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(path); err != nil {
		t.Fatal(err)
	}
	ws, err := a.Adopt(ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(path, "valuable"), []byte("unsaved"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(repo); err != nil {
		t.Fatal(err)
	}
	if err := a.Done(ctx, ws.Path, true); err != nil {
		t.Fatal(err)
	}
	if data, err := os.ReadFile(filepath.Join(path, "valuable")); err != nil || string(data) != "unsaved" {
		t.Fatalf("adopted data lost: %v", err)
	}
	if _, err := gitx.Run(ctx, repo, "show-ref", "--verify", "refs/heads/external"); err != nil {
		t.Fatal("adopted branch deleted")
	}
}
func TestQuotaGCDoesNotDeleteCleanUnpushedCommits(t *testing.T) {
	a, repo := testApp(t)
	ctx := context.Background()
	writeConfig(t, repo, "version: 1\nbase: main\nports: [web]\ngc: {max_workspaces: 1}\n")
	one, err := a.New(ctx, "unpublished", "main", false)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(one.Path, "valuable"), []byte("valuable commit"), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "valuable"}, {"commit", "-m", "unpublished work"}} {
		if _, err := gitx.Run(ctx, one.Path, args...); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := a.New(ctx, "disposable", "main", false); err != nil {
		t.Fatal(err)
	}
	if _, err := a.GC(ctx, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(one.Path, "valuable")); err != nil {
		t.Fatal("GC deleted unpublished work")
	}
	if _, err := gitx.Run(ctx, repo, "show-ref", "--verify", "refs/heads/berth/unpublished"); err != nil {
		t.Fatal("GC deleted unpushed branch")
	}
	if err := a.Done(ctx, one.Path, false); err == nil {
		t.Fatal("done accepted unpublished commits")
	}
}
func TestFailedSetupRetriesUsingWorkspaceConfiguration(t *testing.T) {
	a, repo := testApp(t)
	ctx := context.Background()
	writeConfig(t, repo, "version: 1\nbase: main\nports: [web]\nhooks:\n  setup: ['exit 7']\n")
	if _, err := a.New(ctx, "retry", "main", false); err == nil {
		t.Fatal("setup should fail")
	}
	ws, err := a.resolve(ctx, "retry")
	if err != nil {
		t.Fatal(err)
	}
	if ws.SetupComplete || ws.Phase != "failed" {
		t.Fatalf("failure not recorded: %+v", ws)
	}
	// Fix only the worktree: main deliberately retains the failing configuration.
	if err := os.WriteFile(filepath.Join(ws.Path, "berth.yaml"), []byte("version: 1\nbase: main\nports: [web]\nhooks:\n  setup: ['echo fixed > setup-result']\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := a.New(ctx, "retry", "main", false); err != nil {
		t.Fatal(err)
	}
	fresh, err := a.resolve(ctx, ws.Path)
	if err != nil {
		t.Fatal(err)
	}
	if !fresh.SetupComplete {
		t.Fatal("setup did not recover")
	}
	if _, err := os.Stat(filepath.Join(ws.Path, "setup-result")); err != nil {
		t.Fatal("workspace config was not executed")
	}
}
func TestUnknownProcessStateBlocksReset(t *testing.T) {
	a, _ := testApp(t)
	ctx := context.Background()
	ws, err := a.New(ctx, "preserve", "main", false)
	if err != nil {
		t.Fatal(err)
	}
	file := filepath.Join(ws.Path, ".berth", "data", "valuable")
	if err := os.WriteFile(file, []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	endpoint := process.Socket(ws.Path)
	if runtime.GOOS == "windows" {
		endpoint = process.PortFile(ws.Path)
	}
	if err := os.MkdirAll(filepath.Dir(endpoint), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(endpoint, []byte("stale endpoint, not a live supervisor"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := a.Reset(ctx, ws.Path); err == nil {
		t.Fatal("reset accepted unknown process state")
	}
	if _, err := os.Stat(file); err != nil {
		t.Fatal("data removed despite failed down")
	}
}
func TestGCLeavesLockedWorkspaceAlone(t *testing.T) {
	a, repo := testApp(t)
	ctx := context.Background()
	writeConfig(t, repo, "version: 1\nbase: main\nports: [web]\ngc: {remove_after_days: 1}\n")
	ws, err := a.New(ctx, "active", "main", false)
	if err != nil {
		t.Fatal(err)
	}
	if err := a.Store.Update(ctx, func(f *state.File) error {
		w := f.Workspaces[ws.Path]
		w.LastUsedAt = time.Now().Add(-72 * time.Hour)
		f.Workspaces[ws.Path] = w
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	unlock, err := a.Store.LockWorkspace(ctx, ws.Path)
	if err != nil {
		t.Fatal(err)
	}
	defer unlock()
	rep, err := a.GC(ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Actions) > 0 {
		t.Fatalf("collected active workspace: %+v", rep)
	}
}

func TestFreshWorkspaceCleanAndSafeDone(t *testing.T) {
	a, _ := testApp(t)
	ctx := context.Background()
	ws, err := a.New(ctx, "clean", "main", false)
	if err != nil {
		t.Fatal(err)
	}
	status, err := a.Status(ctx, ws.Path)
	if err != nil {
		t.Fatal(err)
	}
	if status.Dirty {
		t.Fatal("berth metadata dirties a fresh checkout")
	}
	if err := a.Done(ctx, ws.Path, false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(ws.Path); !os.IsNotExist(err) {
		t.Fatal("safe disposable workspace not removed")
	}
}
