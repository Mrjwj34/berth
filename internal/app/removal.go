package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Mrjwj34/lane/internal/gitx"
	"github.com/Mrjwj34/lane/internal/runner"
	"github.com/Mrjwj34/lane/internal/state"
	"github.com/Mrjwj34/lane/internal/worktree"
)

// resumeRemoval only has authority to finish the exact removal recorded after
// teardown and shutdown. Missing checkouts without that record confer no authority.
func (a *App) resumeRemoval(ctx context.Context, ws state.Workspace, force bool) error {
	if ws.Ownership != state.Owned || ws.RemovalHead == "" || ws.Branch != worktree.BranchName(ws.Slug) {
		return fmt.Errorf("invalid workspace removal identity")
	}
	main, err := gitx.MainRepo(ctx, ws.Repo)
	if err != nil {
		return err
	}
	common, err := gitx.CommonDir(ctx, ws.Repo)
	if err != nil {
		return err
	}
	if !worktree.SamePath(main, ws.Repo) || worktree.SamePath(ws.Path, ws.Repo) ||
		!worktree.SamePath(filepath.Dir(ws.GitDir), filepath.Join(common, "worktrees")) {
		return fmt.Errorf("repository identity changed during removal")
	}
	if _, err := os.Lstat(ws.Path); err == nil {
		if err := validateIdentity(ctx, ws); err != nil {
			return err
		}
		head, err := gitx.Run(ctx, ws.Path, "rev-parse", "HEAD")
		if err != nil {
			return err
		}
		if head != ws.RemovalHead {
			return fmt.Errorf("workspace HEAD changed during removal; preserving checkout")
		}
		if !force {
			if err := worktree.Preserved(ctx, ws.Repo, ws.Path, ws.Base); err != nil {
				return err
			}
		}
		if err := runner.Existing(ws).Destroy(ctx); err != nil {
			return a.failure(ws, err)
		}
		if err := worktree.Remove(ctx, ws.Repo, ws.Path, force); err != nil {
			return a.failure(ws, err)
		}
	} else if !os.IsNotExist(err) {
		return err
	}
	// Git's metadata must also be gone: never infer successful worktree removal
	// from an externally deleted/moved directory or prune unrelated metadata.
	if _, err := os.Lstat(ws.GitDir); err == nil {
		return fmt.Errorf("worktree Git metadata remains; finish Git worktree removal before retrying lane done")
	} else if !os.IsNotExist(err) {
		return fmt.Errorf("inspect worktree Git metadata: %w", err)
	}
	if err := runner.Existing(ws).Destroy(ctx); err != nil {
		return a.failure(ws, err)
	}
	if err := worktree.DeleteBranch(ctx, ws.Repo, ws.Branch, ws.RemovalHead); err != nil {
		return a.failure(ws, err)
	}
	return a.unregister(ctx, ws)
}
