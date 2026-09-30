package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Mrjwj34/berth/internal/gitx"
)

func TestFailedResetIsRetriedBeforeUse(t *testing.T) {
	a, repo := testApp(t)
	ctx := context.Background()
	writeConfig(t, repo, "version: 1\nbase: main\nhooks:\n  setup: ['echo initialized > .berth/data/seed']\n")
	v, err := a.New(ctx, "reset-retry", "main", false)
	if err != nil {
		t.Fatal(err)
	}
	data := filepath.Join(v.Path, ".berth", "data")
	if err := os.Remove(filepath.Join(data, "seed")); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(data); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(data, []byte("blocks reset"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := a.Reset(ctx, v.Path); err == nil {
		t.Fatal("reset succeeded with an invalid data directory")
	}
	ws, err := a.resolve(ctx, v.Path)
	if err != nil {
		t.Fatal(err)
	}
	if ws.SetupComplete {
		t.Error("failed reset still marked initialized")
	}
	if err := os.Remove(data); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(data, 0o700); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(data, "partially-reset")
	if err := os.WriteFile(stale, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := a.New(ctx, "reset-retry", "main", false); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Fatal("retry ran setup without finishing data reset")
	}
	if _, err := os.Stat(filepath.Join(data, "seed")); err != nil {
		t.Fatalf("use after failed reset skipped setup: %v", err)
	}
}

func TestDoneRetriesAfterBranchDeletionFailure(t *testing.T) {
	for _, scenario := range []string{"retry", "changed branch", "already deleted", "checked out elsewhere"} {
		t.Run(scenario, func(t *testing.T) {
			a, repo := testApp(t)
			ctx := context.Background()
			v, err := a.New(ctx, "done-retry", "main", false)
			if err != nil {
				t.Fatal(err)
			}
			common, err := gitx.CommonDir(ctx, repo)
			if err != nil {
				t.Fatal(err)
			}
			lock := filepath.Join(common, "refs", "heads", "berth", "done-retry.lock")
			if err := os.WriteFile(lock, nil, 0o600); err != nil {
				t.Fatal(err)
			}
			if err := a.Done(ctx, v.Path, false); err == nil {
				t.Fatal("branch lock should interrupt done")
			}
			if _, err := os.Stat(v.Path); !os.IsNotExist(err) {
				t.Fatalf("expected checkout removal before branch failure: %v", err)
			}
			if err := os.Remove(lock); err != nil {
				t.Fatal(err)
			}
			report, err := a.GC(ctx, false)
			if err != nil || len(report.Actions) != 0 || len(report.Warnings) == 0 {
				t.Fatalf("GC must retain pending removal: %+v, %v", report, err)
			}
			if _, err := a.New(ctx, "done-retry", "main", false); err == nil {
				t.Fatal("new resumed a workspace pending removal")
			}
			if scenario == "changed branch" {
				for _, args := range [][]string{{"commit", "--allow-empty", "-m", "new work"}, {"branch", "-f", v.Branch, "HEAD"}} {
					if _, err := gitx.Run(ctx, repo, args...); err != nil {
						t.Fatal(err)
					}
				}
			}
			if scenario == "already deleted" {
				if _, err := gitx.Run(ctx, repo, "branch", "-D", v.Branch); err != nil {
					t.Fatal(err)
				}
			}
			if scenario == "checked out elsewhere" {
				if _, err := gitx.Run(ctx, repo, "worktree", "add", filepath.Join(t.TempDir(), "other"), v.Branch); err != nil {
					t.Fatal(err)
				}
			}
			err = a.Done(ctx, v.Path, true)
			if scenario == "changed branch" || scenario == "checked out elsewhere" {
				if err == nil {
					t.Fatal("retry deleted a changed branch")
				}
				if _, err := gitx.Run(ctx, repo, "show-ref", "--verify", "refs/heads/"+v.Branch); err != nil {
					t.Fatal("changed branch lost")
				}
				return
			}
			if err != nil {
				t.Fatalf("done could not resume: %v", err)
			}
			f, err := a.Store.Read(ctx)
			if err != nil || len(f.Workspaces) != 0 {
				t.Fatalf("registration not released: %v, %+v", err, f)
			}
			if out, err := gitx.Run(ctx, repo, "branch", "--list", v.Branch); err != nil || out != "" {
				t.Fatalf("branch not removed: %q, %v", out, err)
			}
		})
	}
}

// TestDoneRetriesAfterCheckoutMoved covers an interrupted removal whose
// recorded checkout has since moved: the stale record must not deadlock
// cleanup, and removing must re-derive authority from the live checkout.
func TestDoneRetriesAfterCheckoutMoved(t *testing.T) {
	setup := func(t *testing.T) (*App, string, *WorkspaceView) {
		a, repo := testApp(t)
		ctx := context.Background()
		v, err := a.New(ctx, "moved-checkout", "main", false)
		if err != nil {
			t.Fatal(err)
		}
		// A locked worktree makes `git worktree remove --force` fail after the
		// removal record is written, simulating an interrupted removal.
		if _, err := gitx.Run(ctx, repo, "worktree", "lock", v.Path); err != nil {
			t.Fatal(err)
		}
		if err := a.Done(ctx, v.Path, true); err == nil {
			t.Fatal("locked worktree should interrupt done")
		}
		ws, err := a.resolve(ctx, v.Path)
		if err != nil {
			t.Fatal(err)
		}
		if ws.RemovalHead == "" {
			t.Fatal("interrupted removal did not record the checkout")
		}
		if _, err := gitx.Run(ctx, repo, "worktree", "unlock", v.Path); err != nil {
			t.Fatal(err)
		}
		// Move the checkout: new branch with a tree-changing commit, like the
		// user continuing work in the workspace after the interruption.
		if _, err := gitx.Run(ctx, v.Path, "checkout", "-b", "fix/moved"); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(v.Path, "moved-work"), []byte("unpushed"), 0o644); err != nil {
			t.Fatal(err)
		}
		for _, args := range [][]string{{"add", "moved-work"}, {"commit", "-m", "moved work"}} {
			if _, err := gitx.Run(ctx, v.Path, args...); err != nil {
				t.Fatal(err)
			}
		}
		return a, repo, v
	}
	t.Run("force removes the moved checkout", func(t *testing.T) {
		a, repo, v := setup(t)
		if err := a.Done(context.Background(), v.Path, true); err != nil {
			t.Fatalf("done refused a moved checkout forever: %v", err)
		}
		if _, err := os.Stat(v.Path); !os.IsNotExist(err) {
			t.Fatal("moved checkout not removed")
		}
		// A moved checkout confers no branch deletion authority: both the
		// moved-to branch and the original berth branch keep their commits.
		for _, ref := range []string{"refs/heads/fix/moved", "refs/heads/" + v.Branch} {
			if _, err := gitx.Run(context.Background(), repo, "show-ref", "--verify", ref); err != nil {
				t.Fatalf("branch lost with moved checkout: %s", ref)
			}
		}
		f, err := a.Store.Read(context.Background())
		if err != nil || len(f.Workspaces) != 0 {
			t.Fatalf("registration not released: %v, %+v", err, f)
		}
	})
	t.Run("unpreserved work still blocks removal", func(t *testing.T) {
		a, repo, v := setup(t)
		if err := a.Done(context.Background(), v.Path, false); err == nil {
			t.Fatal("done removed unpreserved work in a moved checkout")
		}
		if _, err := os.Stat(v.Path); err != nil {
			t.Fatal("moved checkout removed despite unpreserved work")
		}
		if _, err := gitx.Run(context.Background(), repo, "show-ref", "--verify", "refs/heads/fix/moved"); err != nil {
			t.Fatal("unpreserved branch lost")
		}
	})
}
