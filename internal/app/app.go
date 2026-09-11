package app

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"time"

	"github.com/Mrjwj34/lane/internal/config"
	"github.com/Mrjwj34/lane/internal/copyfs"
	"github.com/Mrjwj34/lane/internal/gc"
	"github.com/Mrjwj34/lane/internal/gitx"
	"github.com/Mrjwj34/lane/internal/process"
	"github.com/Mrjwj34/lane/internal/runner"
	"github.com/Mrjwj34/lane/internal/state"
	"github.com/Mrjwj34/lane/internal/worktree"
)

type App struct{ Store *state.Store }

func Open(ctx context.Context) (*App, error) {
	st, err := state.Open(ctx)
	if err != nil {
		return nil, err
	}
	return &App{Store: st}, nil
}
func (a *App) Close() error {
	if a == nil || a.Store == nil {
		return nil
	}
	return a.Store.Close()
}

type WorkspaceView struct {
	Slug      string            `json:"slug"`
	Path      string            `json:"path"`
	Repo      string            `json:"repo"`
	Branch    string            `json:"branch"`
	Ports     map[string]int    `json:"ports,omitempty"`
	Listen    map[string]int    `json:"listen,omitempty"`
	Env       map[string]string `json:"env,omitempty"`
	Running   bool              `json:"running"`
	Dirty     bool              `json:"dirty"`
	Processes []process.Proc    `json:"processes,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	LastUsed  time.Time         `json:"last_used_at"`
	Phase     string            `json:"phase,omitempty"`
	Ownership string            `json:"ownership,omitempty"`
	Error     string            `json:"error,omitempty"`
	Runtime   runner.Plan       `json:"runtime"`
}

func portNames(cfg *config.Config) []string {
	names := append([]string{}, cfg.Ports...)
	if runtime.GOOS == "windows" && cfg.Runtime.Kind() == "native" && len(cfg.Processes) > 0 {
		names = append(names, "pc")
	}
	return names
}
func (a *App) New(ctx context.Context, slug, base string, up bool) (*WorkspaceView, error) {
	if err := worktree.ValidateSlug(slug); err != nil {
		return nil, err
	}
	repo, err := gitx.MainRepo(ctx, cwd())
	if err != nil {
		return nil, err
	}
	cfg, _, err := loadConfig(repo)
	if err != nil {
		return nil, err
	}
	if base == "" {
		base = cfg.Base
	}
	path := worktree.ResolvePath(repo, slug, cfg.WorktreeRoot)
	file, err := a.Store.Read(ctx)
	if err != nil {
		return nil, err
	}
	if ws, ok := file.BySlug(repo, slug); ok {
		path = ws.Path
	}
	unlock, err := a.Store.LockWorkspace(ctx, path)
	if err != nil {
		return nil, err
	}
	defer unlock()
	file, err = a.Store.Read(ctx)
	if err != nil {
		return nil, err
	}
	ws, ok := file.BySlug(repo, slug)
	if !ok {
		if _, err := os.Lstat(path); err == nil {
			return nil, fmt.Errorf("existing checkout is not owned by lane; run lane adopt in %s", path)
		} else if !os.IsNotExist(err) {
			return nil, err
		}
		// Preserve orphan checkouts across failures; never guess ownership after a crash.
		if err := worktree.Add(ctx, repo, path, worktree.BranchName(slug), base); err != nil {
			return nil, err
		}
		cfg, _, err = loadConfig(path)
		if err != nil {
			return nil, fmt.Errorf("checkout preserved at %s; fix configuration and adopt: %w", path, err)
		}
		gitDir, err := gitx.Run(ctx, path, "rev-parse", "--absolute-git-dir")
		if err != nil {
			return nil, err
		}
		id, err := newID()
		if err != nil {
			return nil, err
		}
		now := time.Now().UTC()
		ws = state.Workspace{ID: id, Slug: slug, Path: mustKey(path), Repo: mustKey(repo), Branch: worktree.BranchName(slug), Base: base, GitDir: gitDir, Ownership: state.Owned, Phase: "preparing", Runtime: cfg.Runtime, Listen: cfg.Listen, CreatedAt: now, LastUsedAt: now}
		if err := writeIdentity(ws); err != nil {
			return nil, err
		}
		ws, err = a.Store.Reserve(ctx, ws, portNames(cfg))
		if err != nil {
			return nil, err
		}
	}
	s, err := a.session(ctx, ws)
	if err != nil {
		return nil, err
	}
	if !ws.SetupComplete {
		if err := a.prepare(ctx, s, ws); err != nil {
			return nil, a.failure(ws, err)
		}
	}
	if up {
		if err := s.Up(ctx); err != nil {
			return nil, a.failure(ws, err)
		}
		if err := a.record(ctx, ws, "running", nil); err != nil {
			return nil, err
		}
	}
	ws, err = a.resolve(ctx, ws.Path)
	if err != nil {
		return nil, err
	}
	return a.view(ctx, ws, true)
}
func (a *App) Adopt(ctx context.Context, setup bool) (*WorkspaceView, error) {
	top, err := gitx.TopLevel(ctx, cwd())
	if err != nil {
		return nil, err
	}
	repo, err := gitx.MainRepo(ctx, top)
	if err != nil {
		return nil, err
	}
	if err := worktree.ValidateTarget(ctx, repo, top); err != nil {
		return nil, err
	}
	unlock, err := a.Store.LockWorkspace(ctx, top)
	if err != nil {
		return nil, err
	}
	defer unlock()
	cfg, _, err := loadConfig(top)
	if err != nil {
		return nil, err
	}
	file, err := a.Store.Read(ctx)
	if err != nil {
		return nil, err
	}
	ws, ok := file.Lookup(top)
	if !ok || ws.Ownership == "" {
		branch, err := gitx.CurrentBranch(ctx, top)
		if err != nil {
			return nil, err
		}
		gitDir, err := gitx.Run(ctx, top, "rev-parse", "--absolute-git-dir")
		if err != nil {
			return nil, err
		}
		id, err := newID()
		if err != nil {
			return nil, err
		}
		now := time.Now().UTC()
		if !ok {
			ws = state.Workspace{ID: id, Slug: filepath.Base(top), Path: mustKey(top), Repo: mustKey(repo), Branch: branch, Base: cfg.Base, GitDir: gitDir, Ownership: state.Adopted, Phase: "ready", Runtime: cfg.Runtime, Listen: cfg.Listen, CreatedAt: now, LastUsedAt: now, SetupComplete: !setup}
			ws, err = a.Store.Reserve(ctx, ws, portNames(cfg))
			if err != nil {
				return nil, err
			}
		} else {
			if cfg.Runtime.Kind() != "native" {
				return nil, fmt.Errorf("legacy records cannot switch runtime; create a new workspace")
			}
			ws.ID = id
			ws.GitDir = gitDir
			ws.Branch = branch
			ws.Ownership = state.Adopted
			ws.SetupComplete = !setup
			if err := a.Store.Update(ctx, func(f *state.File) error { f.Workspaces[ws.Path] = ws; return nil }); err != nil {
				return nil, err
			}
		}
		if err := writeIdentity(ws); err != nil {
			return nil, err
		}
	}
	s, err := a.session(ctx, ws)
	if err != nil {
		return nil, err
	}
	if setup {
		if err := a.prepare(ctx, s, ws); err != nil {
			return nil, a.failure(ws, err)
		}
	} else if err := a.writeEnv(s); err != nil {
		return nil, err
	}
	ws, err = a.resolve(ctx, ws.Path)
	if err != nil {
		return nil, err
	}
	return a.view(ctx, ws, true)
}
func (a *App) prepare(ctx context.Context, s *runner.Session, ws state.Workspace) error {
	if err := a.update(ctx, ws, func(w *state.Workspace) { w.Phase = "preparing"; w.SetupComplete = false; state.Touch(w) }); err != nil {
		return err
	}
	if err := validateIdentity(ctx, ws); err != nil {
		return err
	}
	if err := worktree.ApplyInclude(ctx, ws.Repo, ws.Path); err != nil {
		return err
	}
	if err := copyfs.CopyDirs(ctx, ws.Repo, ws.Path, s.Config.CopyDirs); err != nil {
		return err
	}
	if err := safeDirectory(ws.Path, ".lane/data"); err != nil {
		return err
	}
	if err := a.writeEnv(s); err != nil {
		return err
	}
	if err := s.Prepare(ctx); err != nil {
		return err
	}
	if err := s.Hooks(ctx, s.Config.Hooks.Setup, os.Stderr, os.Stderr); err != nil {
		return err
	}
	return a.update(ctx, ws, func(w *state.Workspace) { w.Phase = "ready"; w.SetupComplete = true; w.LastError = ""; state.Touch(w) })
}
func (a *App) Up(ctx context.Context, slug string) error {
	return a.locked(ctx, slug, func(ws state.Workspace) error {
		s, err := a.session(ctx, ws)
		if err != nil {
			return err
		}
		if !ws.SetupComplete {
			if err := a.prepare(ctx, s, ws); err != nil {
				return a.failure(ws, err)
			}
		}
		if err := a.writeEnv(s); err != nil {
			return err
		}
		if err := s.Up(ctx); err != nil {
			return a.failure(ws, err)
		}
		return a.record(ctx, ws, "running", nil)
	})
}
func (a *App) Down(ctx context.Context, slug string) error {
	return a.locked(ctx, slug, func(ws state.Workspace) error {
		if err := runner.Existing(ws).Down(ctx); err != nil {
			return a.failure(ws, err)
		}
		return a.record(ctx, ws, "stopped", nil)
	})
}
func (a *App) Run(ctx context.Context, slug string, argv []string) error {
	if len(argv) == 0 {
		return fmt.Errorf("missing command")
	}
	return a.locked(ctx, slug, func(ws state.Workspace) error {
		s, err := a.session(ctx, ws)
		if err != nil {
			return err
		}
		if !ws.SetupComplete {
			if err := a.prepare(ctx, s, ws); err != nil {
				return a.failure(ws, err)
			}
		}
		if err := a.touch(ctx, ws.Path); err != nil {
			return err
		}
		err = s.Run(ctx, argv, os.Stdin, os.Stdout, os.Stderr)
		finish, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		if touchErr := a.touch(finish, ws.Path); err == nil {
			err = touchErr
		}
		return err
	})
}
func (a *App) Reset(ctx context.Context, slug string) error {
	return a.locked(ctx, slug, func(ws state.Workspace) error {
		s, err := a.session(ctx, ws)
		if err != nil {
			return err
		}
		if err := s.Destroy(ctx); err != nil {
			return a.failure(ws, err)
		}
		if err := safeDirectory(ws.Path, ".lane/data"); err != nil {
			return err
		}
		if err := os.RemoveAll(config.DataDir(ws.Path)); err != nil {
			return err
		}
		if err := a.prepare(ctx, s, ws); err != nil {
			return a.failure(ws, err)
		}
		return nil
	})
}
func (a *App) Done(ctx context.Context, slug string, force bool) error {
	return a.locked(ctx, slug, func(ws state.Workspace) error { return a.doneLocked(ctx, ws, force) })
}
func (a *App) doneLocked(ctx context.Context, ws state.Workspace, force bool) error {
	if err := validateIdentity(ctx, ws); err != nil {
		return err
	}
	owned := ws.Ownership == state.Owned
	if owned && !force {
		if err := worktree.Preserved(ctx, ws.Repo, ws.Path, ws.Base); err != nil {
			return err
		}
	}
	s, err := a.session(ctx, ws)
	if err != nil {
		return err
	}
	if err := s.Down(ctx); err != nil {
		return a.failure(ws, err)
	}
	if owned && len(s.Config.Hooks.Teardown) > 0 {
		if err := s.Hooks(ctx, s.Config.Hooks.Teardown, os.Stderr, os.Stderr); err != nil {
			cleanup, cancel := context.WithTimeout(context.Background(), 30*time.Second)
			defer cancel()
			stopErr := s.Down(cleanup)
			if stopErr != nil {
				err = fmt.Errorf("%w; stop failed: %v", err, stopErr)
			}
			return a.failure(ws, err)
		}
	}
	if err := s.Destroy(ctx); err != nil {
		return a.failure(ws, err)
	}
	if owned {
		if err := validateIdentity(ctx, ws); err != nil {
			return err
		}
		if !force {
			if err := worktree.Preserved(ctx, ws.Repo, ws.Path, ws.Base); err != nil {
				return err
			}
		}
		if err := a.record(ctx, ws, "removing", nil); err != nil {
			return err
		}
		if err := worktree.Remove(ctx, ws.Repo, ws.Path, force); err != nil {
			return a.failure(ws, err)
		}
		if err := worktree.DeleteBranch(ctx, ws.Repo, ws.Branch); err != nil {
			return a.failure(ws, err)
		}
	} else {
		// Adopted and migrated worktrees are never owned, even with --force.
		if err := os.Remove(identityPath(ws)); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return a.Store.Update(ctx, func(f *state.File) error {
		if cur, ok := f.Workspaces[ws.Path]; ok && cur.ID == ws.ID {
			delete(f.Workspaces, ws.Path)
		}
		return nil
	})
}
func (a *App) Attach(ctx context.Context, slug string) (*WorkspaceView, error) {
	ws, err := a.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	if err := a.touch(ctx, ws.Path); err != nil {
		return nil, err
	}
	return a.view(ctx, ws, true)
}
func (a *App) Status(ctx context.Context, slug string) (*WorkspaceView, error) {
	ws, err := a.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	return a.view(ctx, ws, true)
}
func (a *App) List(ctx context.Context) ([]WorkspaceView, error) {
	f, err := a.Store.Read(ctx)
	if err != nil {
		return nil, err
	}
	out := []WorkspaceView{}
	for _, ws := range f.Workspaces {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		v, err := a.view(ctx, ws, false)
		if err != nil {
			return nil, err
		}
		out = append(out, *v)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].LastUsed.Equal(out[j].LastUsed) {
			return out[i].Path < out[j].Path
		}
		return out[i].LastUsed.After(out[j].LastUsed)
	})
	return out, nil
}
func (a *App) Logs(ctx context.Context, slug, name string) error {
	if name == "" {
		return fmt.Errorf("process name required")
	}
	ws, err := a.resolve(ctx, slug)
	if err != nil {
		return err
	}
	return runner.Existing(ws).Logs(ctx, name, os.Stdout, os.Stderr)
}
func (a *App) Plan(ctx context.Context, slug string) (runner.Plan, error) {
	ws, err := a.resolve(ctx, slug)
	if err != nil {
		return runner.Plan{}, err
	}
	s, err := a.session(ctx, ws)
	if err != nil {
		return runner.Plan{}, err
	}
	return s.Plan(), nil
}
func (a *App) GC(ctx context.Context, dry bool) (*gc.Report, error) {
	return gc.Run(ctx, a.Store, gc.Options{DryRun: dry, Remove: func(ctx context.Context, ws state.Workspace, force bool) error { return a.doneLocked(ctx, ws, force) }})
}
