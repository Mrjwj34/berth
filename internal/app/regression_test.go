package app

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Mrjwj34/berth/internal/gitx"
)

// A workspace moved to a fix/* branch must stay usable and reclaimable: berth
// identities are path-based, and the registered branch is only the default.
func TestWorkspaceFollowsRenamedBranch(t *testing.T) {
	a, repo := testApp(t)
	ctx := context.Background()
	ws, err := a.New(ctx, "renamed", "main", false)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := gitx.Run(ctx, ws.Path, "checkout", "-B", "fix/renamed"); err != nil {
		t.Fatal(err)
	}
	if _, err := a.Plan(ctx, ws.Path); err != nil {
		t.Fatalf("plan rejected a renamed branch: %v", err)
	}
	view, err := a.Status(ctx, ws.Path)
	if err != nil {
		t.Fatal(err)
	}
	if view.Branch != "fix/renamed" {
		t.Fatalf("status branch = %q, want the current checkout", view.Branch)
	}
	if err := a.Done(ctx, ws.Path, true); err != nil {
		t.Fatalf("done rejected a renamed branch: %v", err)
	}
	file, err := a.Store.Read(ctx)
	if err != nil || len(file.Workspaces) != 0 {
		t.Fatalf("registration not released: %v, %+v", err, file)
	}
	if _, err := os.Stat(ws.Path); !os.IsNotExist(err) {
		t.Fatalf("checkout not removed: %v", err)
	}
	if _, err := gitx.Run(ctx, repo, "show-ref", "--verify", "refs/heads/fix/renamed"); err != nil {
		t.Fatal("user-owned fix/* branch was deleted")
	}
}

// A branch-local contract change keeps refusing in-place up, but it must never
// strand the registration: done --force reclaims it.
func TestDoneReclaimsChangedContract(t *testing.T) {
	a, _ := testApp(t)
	ctx := context.Background()
	ws, err := a.New(ctx, "contract", "main", false)
	if err != nil {
		t.Fatal(err)
	}
	changed := "version: 1\nbase: main\nports: [web, api]\n"
	if err := os.WriteFile(filepath.Join(ws.Path, "berth.yaml"), []byte(changed), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := a.Up(ctx, ws.Path); err == nil || !strings.Contains(err.Error(), "contract") {
		t.Fatalf("up accepted a changed contract: %v", err)
	}
	if err := a.Done(ctx, ws.Path, true); err != nil {
		t.Fatalf("done could not reclaim a changed contract: %v", err)
	}
	file, err := a.Store.Read(ctx)
	if err != nil || len(file.Workspaces) != 0 {
		t.Fatalf("registration not released: %v, %+v", err, file)
	}
}

// GC must offer contract-changed workspaces as candidates once their commits
// are preserved, without ever forcing removal.
func TestGCReclaimsChangedContract(t *testing.T) {
	a, repo := testApp(t)
	ctx := context.Background()
	ws, err := a.New(ctx, "gc-contract", "main", false)
	if err != nil {
		t.Fatal(err)
	}
	changed := "version: 1\nbase: main\nports: [web, api]\n"
	if err := os.WriteFile(filepath.Join(ws.Path, "berth.yaml"), []byte(changed), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{{"add", "berth.yaml"}, {"commit", "-m", "change contract"}} {
		if _, err := gitx.Run(ctx, ws.Path, args...); err != nil {
			t.Fatal(err)
		}
	}
	// Preserve the work by advancing the base to the workspace commit.
	if _, err := gitx.Run(ctx, repo, "merge", "--ff-only", "berth/gc-contract"); err != nil {
		t.Fatal(err)
	}
	report, err := a.GC(ctx, false)
	if err != nil {
		t.Fatal(err)
	}
	found := false
	for _, action := range report.Actions {
		if action.Kind == "reclaim_contract" {
			found = true
		}
	}
	if !found {
		t.Fatalf("gc did not list the contract change: %+v", report)
	}
	file, err := a.Store.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := file.Lookup(ws.Path); ok {
		t.Fatal("registration remains after gc")
	}
}
