package worktree

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Mrjwj34/berth/internal/gitx"
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
		if SamePath(info.Path, path) {
			found = true
			if info.Branch != "berth/feat-x" {
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
	run("config", "user.email", "berth@test")
	run("config", "user.name", "berth")
	if err := os.WriteFile(filepath.Join(dir, "README"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "README")
	run("commit", "-m", "init")
	return dir
}

// Preserved must accept work that reached the base as a squash or cherry-pick,
// where the original commit is not an ancestor.
func TestMergedIntoAcceptsSquashAndCherryPick(t *testing.T) {
	ctx := context.Background()
	run := func(t *testing.T, dir string, args ...string) string {
		t.Helper()
		out, err := gitx.Run(ctx, dir, args...)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	write := func(t *testing.T, dir, name, body string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	t.Run("multiple-commit squash shares the base tree", func(t *testing.T) {
		repo := initRepo(t)
		run(t, repo, "checkout", "-b", "feature")
		write(t, repo, "feature.txt", "one\n")
		run(t, repo, "add", "feature.txt")
		run(t, repo, "commit", "-m", "feature one")
		write(t, repo, "feature.txt", "two\n")
		run(t, repo, "add", "feature.txt")
		run(t, repo, "commit", "-m", "feature two")
		head := run(t, repo, "rev-parse", "HEAD")
		run(t, repo, "checkout", "main")
		run(t, repo, "merge", "--squash", "feature")
		run(t, repo, "commit", "-m", "squash: feature (#1)")
		merged, err := MergedInto(ctx, repo, head, "main")
		if err != nil {
			t.Fatal(err)
		}
		if !merged {
			t.Fatal("squash merge not detected as preserved")
		}
	})
	t.Run("cherry-pick is patch-equivalent after the base moves on", func(t *testing.T) {
		repo := initRepo(t)
		run(t, repo, "checkout", "-b", "feature")
		write(t, repo, "feature.txt", "one\n")
		run(t, repo, "add", "feature.txt")
		run(t, repo, "commit", "-m", "feature one")
		head := run(t, repo, "rev-parse", "HEAD")
		run(t, repo, "checkout", "main")
		run(t, repo, "cherry-pick", head)
		write(t, repo, "later.txt", "later\n")
		run(t, repo, "add", "later.txt")
		run(t, repo, "commit", "-m", "later base work")
		merged, err := MergedInto(ctx, repo, head, "main")
		if err != nil {
			t.Fatal(err)
		}
		if !merged {
			t.Fatal("cherry-picked commit not detected as preserved")
		}
	})
}

// A narrow remote.origin.fetch leaves branch.<name>.remote set without
// origin/<name>, so @{upstream} fails even though the branch is pushed.
func TestUnpublishedFallsBackToRemoteTrackingRef(t *testing.T) {
	ctx := context.Background()
	repo := initRepo(t)
	run := func(args ...string) string {
		t.Helper()
		out, err := gitx.Run(ctx, repo, args...)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	run("remote", "add", "origin", repo)
	run("update-ref", "refs/remotes/origin/main", run("rev-parse", "HEAD"))
	unpublished, why, err := Unpublished(ctx, repo)
	if err != nil {
		t.Fatal(err)
	}
	if unpublished {
		t.Fatalf("remote-tracking fallback missed: %s", why)
	}
	if err := os.WriteFile(filepath.Join(repo, "extra.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	run("add", "extra.txt")
	run("commit", "-m", "local only")
	unpublished, why, err = Unpublished(ctx, repo)
	if err != nil {
		t.Fatal(err)
	}
	if !unpublished || !strings.Contains(why, "origin/main") {
		t.Fatalf("unpublished commit not reported: %v, %q", unpublished, why)
	}
}

// A narrow remote.<name>.fetch leaves origin/<base> absent even though it exists
// on the remote; Add must fetch that exact ref instead of failing.
func TestAddFetchesMissingRemoteTrackingBase(t *testing.T) {
	ctx := context.Background()
	runIn := func(dir string, args ...string) string {
		t.Helper()
		out, err := gitx.Run(ctx, dir, args...)
		if err != nil {
			t.Fatal(err)
		}
		return out
	}
	origin := t.TempDir()
	runIn(origin, "init", "--bare", "-b", "main")
	seed := t.TempDir()
	runIn(seed, "init", "-b", "main")
	runIn(seed, "config", "user.email", "berth@test")
	runIn(seed, "config", "user.name", "berth")
	if err := os.WriteFile(filepath.Join(seed, "README"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	runIn(seed, "add", "README")
	runIn(seed, "commit", "-m", "init")
	runIn(seed, "remote", "add", "origin", origin)
	runIn(seed, "push", "origin", "main")

	repo := filepath.Join(t.TempDir(), "clone")
	runIn(filepath.Dir(repo), "clone", origin, repo)
	runIn(repo, "config", "remote.origin.fetch", "+refs/heads/none:refs/remotes/origin/none")
	runIn(repo, "update-ref", "-d", "refs/remotes/origin/main")
	if _, err := gitx.Run(ctx, repo, "rev-parse", "--verify", "refs/remotes/origin/main"); err == nil {
		t.Fatal("origin/main unexpectedly present before Add")
	}
	if err := Add(ctx, repo, DefaultPath(repo, "fetch-base"), BranchName("fetch-base"), "origin/main"); err != nil {
		t.Fatalf("Add did not fetch a missing remote-tracking base: %v", err)
	}
	if _, err := gitx.Run(ctx, repo, "rev-parse", "--verify", "refs/remotes/origin/main"); err != nil {
		t.Fatal("origin/main was not created by the fetch")
	}
}
