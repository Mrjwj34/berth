package app

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/Mrjwj34/berth/internal/config"
	"github.com/Mrjwj34/berth/internal/gitx"
	"github.com/Mrjwj34/berth/internal/runner"
	"github.com/Mrjwj34/berth/internal/state"
	"github.com/Mrjwj34/berth/internal/worktree"
)

func cwd() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	return dir
}
func loadConfig(root string) (*config.Config, string, error) {
	path := filepath.Join(root, config.Filename)
	if _, err := os.Stat(path); os.IsNotExist(err) {
		d := config.Defaults()
		return &d, root, nil
	} else if err != nil {
		return nil, root, err
	}
	cfg, err := config.Load(path)
	return cfg, root, err
}
func mustKey(path string) string {
	key, err := state.Key(path)
	if err != nil {
		return filepath.Clean(path)
	}
	return key
}
func newID() (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return fmt.Sprintf("%x", b[:]), nil
}
func (a *App) resolve(ctx context.Context, key string) (state.Workspace, error) {
	f, err := a.Store.Read(ctx)
	if err != nil {
		return state.Workspace{}, err
	}
	if key != "" {
		if ws, ok := f.Lookup(key); ok {
			return ws, nil
		}
		if repo, err := gitx.MainRepo(ctx, cwd()); err == nil {
			if ws, ok := f.BySlug(repo, key); ok {
				return ws, nil
			}
		}
		var matches []state.Workspace
		for _, ws := range f.Workspaces {
			if ws.Slug == key {
				matches = append(matches, ws)
			}
		}
		if len(matches) == 1 {
			return matches[0], nil
		}
		if len(matches) > 1 {
			return state.Workspace{}, fmt.Errorf("ambiguous slug %q; use workspace path", key)
		}
	} else {
		here := mustKey(cwd())
		var best state.Workspace
		for path, ws := range f.Workspaces {
			if (here == path || strings.HasPrefix(here, path+string(filepath.Separator))) && len(path) > len(best.Path) {
				best = ws
			}
		}
		if best.Path != "" {
			return best, nil
		}
	}
	return state.Workspace{}, fmt.Errorf("workspace not found; run berth ls or berth adopt")
}
func (a *App) locked(ctx context.Context, key string, fn func(state.Workspace) error) error {
	ws, err := a.resolve(ctx, key)
	if err != nil {
		return err
	}
	unlock, err := a.Store.LockWorkspace(ctx, ws.Path)
	if err != nil {
		return err
	}
	defer unlock()
	cur, err := a.resolve(ctx, ws.Path)
	if err != nil {
		return err
	}
	if cur.ID != ws.ID {
		return fmt.Errorf("workspace replaced during operation")
	}
	return fn(cur)
}
func (a *App) update(ctx context.Context, ws state.Workspace, fn func(*state.Workspace)) error {
	return a.Store.Update(ctx, func(f *state.File) error {
		cur, ok := f.Workspaces[ws.Path]
		if !ok || cur.ID != ws.ID {
			return fmt.Errorf("workspace identity changed")
		}
		fn(&cur)
		f.Workspaces[ws.Path] = cur
		return nil
	})
}
func (a *App) touch(ctx context.Context, path string) error {
	ws, err := a.resolve(ctx, path)
	if err != nil {
		return err
	}
	return a.update(ctx, ws, state.Touch)
}
func (a *App) record(ctx context.Context, ws state.Workspace, phase string, cause error) error {
	return a.update(ctx, ws, func(w *state.Workspace) {
		w.Phase = phase
		w.LastError = ""
		if cause != nil {
			w.LastError = cause.Error()
		}
		state.Touch(w)
	})
}
func (a *App) failure(ws state.Workspace, err error) error {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if e := a.record(ctx, ws, "failed", err); e != nil {
		return fmt.Errorf("%w (record state: %v)", err, e)
	}
	return err
}
func identityPath(ws state.Workspace) string {
	return filepath.Join(ws.Path, ".berth", "identity.json")
}
func safeDirectory(root, rel string) error {
	cur := root
	for _, part := range strings.Split(filepath.ToSlash(rel), "/") {
		if part == "" || part == "." || part == ".." {
			return fmt.Errorf("invalid managed directory")
		}
		cur = filepath.Join(cur, part)
		st, err := os.Lstat(cur)
		if os.IsNotExist(err) {
			if err := os.Mkdir(cur, 0o700); err != nil && !os.IsExist(err) {
				return err
			}
			st, err = os.Lstat(cur)
		}
		if err != nil {
			return err
		}
		if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("managed directory is not a real directory: %s", cur)
		}
	}
	return nil
}
func writeIdentity(ws state.Workspace) error {
	if err := safeDirectory(ws.Path, ".berth/data"); err != nil {
		return err
	}
	for _, name := range []string{"identity.json", ".gitignore"} {
		if st, err := os.Lstat(filepath.Join(ws.Path, ".berth", name)); err == nil && !st.Mode().IsRegular() {
			return fmt.Errorf("invalid metadata file: %s", name)
		}
	}
	if err := os.WriteFile(filepath.Join(ws.Path, ".berth", ".gitignore"), []byte("*\n"), 0o600); err != nil {
		return err
	}
	data, err := json.Marshal(struct{ ID, Repo, GitDir string }{ws.ID, ws.Repo, ws.GitDir})
	if err != nil {
		return err
	}
	return os.WriteFile(identityPath(ws), append(data, '\n'), 0o600)
}
func validateIdentity(ctx context.Context, ws state.Workspace) error {
	if err := worktree.ValidateTarget(ctx, ws.Repo, ws.Path); err != nil {
		return err
	}
	if ws.ID == "" || ws.GitDir == "" || ws.Ownership == "" {
		return fmt.Errorf("legacy workspace: run berth adopt in it to register safe lifecycle identity")
	}
	dir, err := gitx.Run(ctx, ws.Path, "rev-parse", "--absolute-git-dir")
	if err != nil {
		return err
	}
	if !worktree.SamePath(dir, ws.GitDir) {
		return fmt.Errorf("worktree Git identity changed")
	}
	branch, err := gitx.CurrentBranch(ctx, ws.Path)
	if err != nil {
		return err
	}
	if branch != ws.Branch {
		return fmt.Errorf("workspace branch changed from %s to %s", ws.Branch, branch)
	}
	st, err := os.Lstat(filepath.Join(ws.Path, ".berth"))
	if err != nil {
		return err
	}
	if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("invalid .berth directory")
	}
	st, err = os.Lstat(identityPath(ws))
	if err != nil {
		return err
	}
	if !st.Mode().IsRegular() {
		return fmt.Errorf("invalid identity marker")
	}
	data, err := os.ReadFile(identityPath(ws))
	if err != nil {
		return err
	}
	var m struct{ ID, Repo, GitDir string }
	if err := json.Unmarshal(data, &m); err != nil {
		return err
	}
	if m.ID != ws.ID || !worktree.SamePath(m.Repo, ws.Repo) || !worktree.SamePath(m.GitDir, ws.GitDir) {
		return fmt.Errorf("workspace identity marker mismatch")
	}
	return nil
}
func (a *App) session(ctx context.Context, ws state.Workspace) (*runner.Session, error) {
	if ws.RemovalHead != "" {
		return nil, fmt.Errorf("workspace removal is pending; retry berth done %s", ws.Path)
	}
	if err := validateIdentity(ctx, ws); err != nil {
		return nil, err
	}
	cfg, _, err := loadConfig(ws.Path)
	if err != nil {
		return nil, err
	}
	return runner.New(cfg, ws)
}
func (a *App) writeEnv(s *runner.Session) error {
	rel := s.Config.EnvFile
	if rel == "" {
		return nil
	}
	if err := config.RelativePath(rel); err != nil {
		return err
	}
	if parent := filepath.Dir(rel); parent != "." {
		if err := safeDirectory(s.Workspace.Path, filepath.ToSlash(parent)); err != nil {
			return err
		}
	}
	path := filepath.Join(s.Workspace.Path, rel)
	if st, err := os.Lstat(path); err == nil && !st.Mode().IsRegular() {
		return fmt.Errorf("env_file must be a regular file")
	}
	return config.WriteEnvFile(path, s.Env())
}
func (a *App) view(ctx context.Context, ws state.Workspace, withEnv bool) (*WorkspaceView, error) {
	v := &WorkspaceView{Slug: ws.Slug, Path: ws.Path, Repo: ws.Repo, Branch: ws.Branch, Ports: ws.Ports, Listen: ws.Listen, CreatedAt: ws.CreatedAt, LastUsed: ws.LastUsedAt, Phase: ws.Phase, Ownership: ws.Ownership, Error: ws.LastError}
	cfg, _, err := loadConfig(ws.Path)
	if err != nil {
		v.Error = err.Error()
	}
	s := &runner.Session{Config: cfg, Workspace: ws}
	v.Runtime = s.Plan()
	if withEnv {
		v.Env = s.HostEnv()
	}
	dirty, _, err := worktree.Dirty(ctx, ws.Path)
	v.Dirty = dirty
	if err != nil {
		v.Error = err.Error()
	}
	running, err := s.Running(ctx)
	v.Running = running
	if err != nil {
		v.Error = err.Error()
	}
	if running {
		procs, err := s.Status(ctx)
		if err != nil {
			v.Error = err.Error()
		} else {
			v.Processes = procs
		}
	}
	return v, nil
}
