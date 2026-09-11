package gc

import (
	"context"
	"fmt"
	"os"
	"sort"
	"time"

	"github.com/Mrjwj34/lane/internal/config"
	"github.com/Mrjwj34/lane/internal/process"
	"github.com/Mrjwj34/lane/internal/state"
	"github.com/Mrjwj34/lane/internal/worktree"
)

type Action struct {
	Kind   string `json:"kind"`
	Path   string `json:"path"`
	Slug   string `json:"slug,omitempty"`
	Reason string `json:"reason"`
}

type Report struct {
	Actions []Action `json:"actions"`
}

type Options struct {
	DryRun bool
	Now    time.Time
	Down   func(ctx context.Context, path string) error
	Remove func(ctx context.Context, ws state.Workspace, force bool) error
}

func Run(ctx context.Context, st *state.Store, opts Options) (*Report, error) {
	if opts.Now.IsZero() {
		opts.Now = time.Now().UTC()
	}
	if opts.Down == nil {
		opts.Down = process.Down
	}
	file, err := st.Read(ctx)
	if err != nil {
		return nil, err
	}
	rep := &Report{}

	// 1. vanished directories
	for key, ws := range file.Workspaces {
		if _, err := os.Stat(ws.Path); os.IsNotExist(err) {
			rep.Actions = append(rep.Actions, Action{Kind: "reclaim", Path: ws.Path, Slug: ws.Slug, Reason: "directory is gone"})
			if opts.DryRun {
				continue
			}
			_ = opts.Down(ctx, ws.Path)
			if err := st.Update(ctx, func(f *state.File) error {
				delete(f.Workspaces, key)
				return nil
			}); err != nil {
				return rep, err
			}
		}
	}

	file, err = st.Read(ctx)
	if err != nil {
		return nil, err
	}

	// 2. idle stop
	for _, ws := range file.Workspaces {
		cfg, _, err := config.Find(ws.Repo)
		if err != nil || cfg.GC.IdleStopHours <= 0 {
			continue
		}
		if !process.Running(ctx, ws.Path) {
			continue
		}
		idle := opts.Now.Sub(ws.LastUsedAt)
		if ws.LastUsedAt.IsZero() || idle < time.Duration(cfg.GC.IdleStopHours)*time.Hour {
			continue
		}
		rep.Actions = append(rep.Actions, Action{Kind: "idle_stop", Path: ws.Path, Slug: ws.Slug, Reason: fmt.Sprintf("idle for %s", idle.Round(time.Minute))})
		if !opts.DryRun {
			if err := opts.Down(ctx, ws.Path); err != nil {
				return rep, err
			}
		}
	}

	// 3. merged branch cleanup
	for _, ws := range file.Workspaces {
		cfg, _, err := config.Find(ws.Repo)
		if err != nil || cfg.GC.RemoveAfterDays <= 0 {
			continue
		}
		base := ws.Base
		if base == "" {
			base = cfg.Base
		}
		merged, err := worktree.MergedInto(ctx, ws.Repo, ws.Branch, base)
		if err != nil || !merged {
			continue
		}
		age := opts.Now.Sub(ws.LastUsedAt)
		if ws.LastUsedAt.IsZero() {
			age = opts.Now.Sub(ws.CreatedAt)
		}
		if age < time.Duration(cfg.GC.RemoveAfterDays)*24*time.Hour {
			continue
		}
		dirty, _, _ := worktree.Dirty(ctx, ws.Path)
		if dirty {
			continue
		}
		rep.Actions = append(rep.Actions, Action{Kind: "remove_merged", Path: ws.Path, Slug: ws.Slug, Reason: fmt.Sprintf("branch %s merged into %s", ws.Branch, base)})
		if !opts.DryRun && opts.Remove != nil {
			if err := opts.Remove(ctx, ws, false); err != nil {
				return rep, err
			}
		}
	}

	file, err = st.Read(ctx)
	if err != nil {
		return nil, err
	}

	// 4. quota eviction per repo
	byRepo := map[string][]state.Workspace{}
	for _, ws := range file.Workspaces {
		byRepo[ws.Repo] = append(byRepo[ws.Repo], ws)
	}
	for repo, list := range byRepo {
		cfg, _, err := config.Find(repo)
		if err != nil || cfg.GC.MaxWorkspaces <= 0 || len(list) <= cfg.GC.MaxWorkspaces {
			continue
		}
		sort.Slice(list, func(i, j int) bool {
			return list[i].LastUsedAt.Before(list[j].LastUsedAt)
		})
		overflow := len(list) - cfg.GC.MaxWorkspaces
		for _, ws := range list {
			if overflow <= 0 {
				break
			}
			if process.Running(ctx, ws.Path) {
				continue
			}
			dirty, _, _ := worktree.Dirty(ctx, ws.Path)
			if dirty {
				continue
			}
			rep.Actions = append(rep.Actions, Action{Kind: "evict", Path: ws.Path, Slug: ws.Slug, Reason: fmt.Sprintf("over max_workspaces=%d", cfg.GC.MaxWorkspaces)})
			overflow--
			if !opts.DryRun && opts.Remove != nil {
				if err := opts.Remove(ctx, ws, false); err != nil {
					return rep, err
				}
			}
		}
	}

	return rep, nil
}
