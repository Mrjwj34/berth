package worktree

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/Mrjwj34/lane/internal/gitx"
)

func TestValidateSlug(t *testing.T) {
	if err := ValidateSlug("feat-login"); err != nil {
		t.Fatal(err)
	}
	if err := ValidateSlug("Bad"); err == nil {
		t.Fatal("expected error")
	}
}

func TestAddListRemove(t *testing.T) {
	ctx := context.Background()
	repo := initRepo(t)
	path := DefaultPath(repo, "feat-x")
	if err := Add(ctx, repo, path, BranchName("feat-x"), "main"); err != nil {
		t.Fatal(err)
	}
	infos, err := List(ctx, repo)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, info := range infos {
		if info.Path == path {
			found = true
			if info.Branch != "lane/feat-x" {
				t.Fatalf("branch = %s", info.Branch)
			}
		}
	}
	if !found {
		t.Fatalf("worktree not listed: %+v", infos)
	}
	if err := os.WriteFile(filepath.Join(repo, ".worktreeinclude"), []byte("# c\n.extra\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(repo, ".extra"), []byte("secret"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := ApplyInclude(ctx, repo, path); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(path, ".extra"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "secret" {
		t.Fatalf("include copy = %q", got)
	}
	if err := Remove(ctx, repo, path, true); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("path still exists: %v", err)
	}
}

func initRepo(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	run := func(args ...string) {
		t.Helper()
		if _, err := gitx.Run(context.Background(), dir, args...); err != nil {
			t.Fatal(err)
		}
	}
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	run("init", "-b", "main")
	run("config", "user.email", "lane@test")
	run("config", "user.name", "lane")
	if err := os.WriteFile(filepath.Join(dir, "README"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "README")
	run("commit", "-m", "init")
	return dir
}
