package app

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/Mrjwj34/lane/internal/gitx"
)

func TestFailedResetIsRetriedBeforeUse(t *testing.T) {
	a, repo := testApp(t)
	ctx := context.Background()
	writeConfig(t, repo, "version: 1\nbase: main\nhooks:\n  setup: ['echo initialized > .lane/data/seed']\n")
	v, err := a.New(ctx, "reset-retry", "main", false)
	if err != nil {
		t.Fatal(err)
	}
	data := filepath.Join(v.Path, ".lane", "data")
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
			lock := filepath.Join(common, "refs", "heads", "lane", "done-retry.lock")
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
