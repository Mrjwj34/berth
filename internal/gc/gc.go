// Package gc performs explicit, conservative collection under lifecycle locks.
package gc

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"

	"github.com/Mrjwj34/lane/internal/config"
	"github.com/Mrjwj34/lane/internal/runner"
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
	Actions  []Action `json:"actions"`
	Warnings []string `json:"warnings,omitempty"`
}
type Options struct {
	DryRun bool
	Now    time.Time
	Down   func(context.Context, string) error
	Remove func(context.Context, state.Workspace, bool) error
}

func Run(ctx context.Context, st *state.Store, opts Options) (*Report, error) {
	if opts.Now.IsZero() {
		opts.Now = time.Now().UTC()
	}
	f, err := st.Read(ctx)
	if err != nil {
		return nil, err
	}
	rep := &Report{Actions: []Action{}}
	counts := map[string]int{}
	var list []state.Workspace
	for _, w := range f.Workspaces {
		list = append(list, w)
		if w.Ownership == state.Owned {
			counts[w.Repo]++
		}
	}
	sort.Slice(list, func(i, j int) bool {
		if list[i].LastUsedAt.Equal(list[j].LastUsedAt) {
			return list[i].Path < list[j].Path
		}
		return list[i].LastUsedAt.Before(list[j].LastUsedAt)
	})
	for _, snapshot := range list {
		if err := ctx.Err(); err != nil {
			return rep, err
		}
		short, cancel := context.WithTimeout(ctx, 30*time.Millisecond)
		unlock, err := st.LockWorkspace(short, snapshot.Path)
		cancel()
		if err != nil {
			if ctx.Err() != nil {
				return rep, ctx.Err()
			}
			continue
		}
		err = collect(ctx, st, opts, rep, snapshot, counts)
		unlock()
		if err != nil {
			rep.Warnings = append(rep.Warnings, fmt.Sprintf("%s: %v", snapshot.Path, err))
		}
	}
	return rep, nil
}
func collect(ctx context.Context, st *state.Store, opts Options, rep *Report, snapshot state.Workspace, counts map[string]int) error {
	f, err := st.Read(ctx)
	if err != nil {
		return err
	}
	w, ok := f.Workspaces[snapshot.Path]
	if !ok || w.ID != snapshot.ID || !w.LastUsedAt.Equal(snapshot.LastUsedAt) {
		return nil
	}
	if w.RemovalHead != "" {
		return fmt.Errorf("unfinished removal retained; retry lane done %s", w.Path)
	}
	runtime := runner.Existing(w)
	down := func() error {
		if opts.Down != nil {
			return opts.Down(ctx, w.Path)
		}
		return runtime.Down(ctx)
	}
	add := func(kind, reason string) {
		rep.Actions = append(rep.Actions, Action{Kind: kind, Path: w.Path, Slug: w.Slug, Reason: reason})
	}
	if _, err := os.Stat(w.Path); os.IsNotExist(err) {
		if !opts.DryRun {
			if err := down(); err != nil {
				return err
			}
			if err := runtime.Destroy(ctx); err != nil {
				return err
			}
			if err := st.Update(ctx, func(f *state.File) error {
				cur, ok := f.Workspaces[w.Path]
				if ok && cur.ID == w.ID {
					delete(f.Workspaces, w.Path)
				}
				return nil
			}); err != nil {
				return err
			}
		}
		if w.Ownership == state.Owned {
			counts[w.Repo]--
		}
		add("reclaim", "directory gone; runtime stopped before releasing registration")
		return nil
	} else if err != nil {
		return err
	}
	cfg := config.Defaults()
	if _, err := os.Stat(filepath.Join(w.Path, config.Filename)); err == nil {
		loaded, err := config.Load(filepath.Join(w.Path, config.Filename))
		if err != nil {
			return err
		}
		cfg = *loaded
	} else if !os.IsNotExist(err) {
		return err
	}
	running, err := runtime.Running(ctx)
	if err != nil {
		return err
	}
	if running && !w.LastUsedAt.IsZero() && cfg.GC.IdleStopHours > 0 && opts.Now.Sub(w.LastUsedAt) >= time.Duration(cfg.GC.IdleStopHours)*time.Hour {
		if !opts.DryRun {
			if err := down(); err != nil {
				return err
			}
		}
		add("idle_stop", "no lane operation within configured idle period")
		return nil
	}
	if running || w.Ownership != state.Owned || !w.SetupComplete {
		return nil
	}
	age := opts.Now.Sub(w.LastUsedAt)
	if w.LastUsedAt.IsZero() {
		age = opts.Now.Sub(w.CreatedAt)
	}
	kind, reason := "", ""
	if cfg.GC.RemoveAfterDays > 0 && age >= time.Duration(cfg.GC.RemoveAfterDays)*24*time.Hour {
		merged, err := worktree.MergedInto(ctx, w.Repo, w.Branch, w.Base)
		if err != nil {
			return err
		}
		if merged {
			kind = "remove_merged"
			reason = "merged branch beyond retention"
		}
	}
	if kind == "" && cfg.GC.MaxWorkspaces > 0 && counts[w.Repo] > cfg.GC.MaxWorkspaces {
		kind = "evict"
		reason = "over quota; commits preserved"
	}
	if kind == "" {
		return nil
	}
	if err := worktree.Preserved(ctx, w.Repo, w.Path, w.Base); err != nil {
		return nil
	}
	if !opts.DryRun {
		if opts.Remove == nil {
			return fmt.Errorf("safe removal callback missing")
		}
		// Automatic collection must never grant force authority.
		if err := opts.Remove(ctx, w, false); err != nil {
			return err
		}
	}
	counts[w.Repo]--
	add(kind, reason)
	return nil
}
