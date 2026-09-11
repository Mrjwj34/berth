package app

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/Mrjwj34/lane/internal/config"
	"github.com/Mrjwj34/lane/internal/copyfs"
	"github.com/Mrjwj34/lane/internal/gc"
	"github.com/Mrjwj34/lane/internal/gitx"
	"github.com/Mrjwj34/lane/internal/home"
	"github.com/Mrjwj34/lane/internal/process"
	"github.com/Mrjwj34/lane/internal/skill"
	"github.com/Mrjwj34/lane/internal/state"
	"github.com/Mrjwj34/lane/internal/worktree"
)

type App struct {
	Store *state.Store
}

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
	Env       map[string]string `json:"env,omitempty"`
	Running   bool              `json:"running"`
	Dirty     bool              `json:"dirty"`
	Processes []process.Proc    `json:"processes,omitempty"`
	CreatedAt time.Time         `json:"created_at"`
	LastUsed  time.Time         `json:"last_used_at"`
}

func (a *App) New(ctx context.Context, slug, base string, up bool) (*WorkspaceView, error) {
	if err := worktree.ValidateSlug(slug); err != nil {
		return nil, err
	}
	repo, err := gitx.MainRepo(ctx, cwd())
	if err != nil {
		return nil, fmt.Errorf("not a git repository. Run lane from a repo or clone one first: %w", err)
	}
	cfg, _, err := loadConfig(repo)
	if err != nil {
		return nil, err
	}
	if base == "" {
		base = cfg.Base
		if base == "" {
			base = gitx.DefaultBranch(ctx, repo)
		}
	}
	existing, ok := a.lookupSlug(ctx, repo, slug)
	if ok {
		if up {
			if err := a.Up(ctx, existing.Path); err != nil {
				return nil, err
			}
		}
		return a.view(ctx, existing, true)
	}
	path := worktree.ResolvePath(repo, slug, cfg.WorktreeRoot)
	branch := worktree.BranchName(slug)
	if err := worktree.Add(ctx, repo, path, branch, base); err != nil {
		return nil, err
	}
	if err := worktree.ApplyInclude(ctx, repo, path); err != nil {
		return nil, err
	}
	if err := copyfs.CopyDirs(ctx, repo, path, cfg.CopyDirs); err != nil {
		return nil, err
	}
	if err := os.MkdirAll(config.DataDir(path), 0o755); err != nil {
		return nil, err
	}

	portNames := append([]string{}, cfg.Ports...)
	if runtime.GOOS == "windows" && len(cfg.Processes) > 0 {
		portNames = append(portNames, "pc")
	}
	allocated, err := a.allocate(ctx, portNames)
	if err != nil {
		return nil, err
	}

	ws := state.Workspace{
		ID:         newID(),
		Slug:       slug,
		Path:       mustKey(path),
		Repo:       mustKey(repo),
		Branch:     branch,
		Base:       base,
		Ports:      allocated,
		CreatedAt:  time.Now().UTC(),
		LastUsedAt: time.Now().UTC(),
	}
	if err := a.Store.Update(ctx, func(f *state.File) error {
		f.Workspaces[ws.Path] = ws
		return nil
	}); err != nil {
		return nil, err
	}
	if err := a.prepare(ctx, cfg, ws); err != nil {
		return nil, err
	}
	if up {
		if err := a.Up(ctx, ws.Path); err != nil {
			return nil, err
		}
	}
	return a.view(ctx, ws, true)
}

func (a *App) Adopt(ctx context.Context, setup bool) (*WorkspaceView, error) {
	top, err := gitx.TopLevel(ctx, cwd())
	if err != nil {
		return nil, fmt.Errorf("cwd is not a git worktree. cd into the worktree first: %w", err)
	}
	repo, err := gitx.MainRepo(ctx, top)
	if err != nil {
		return nil, err
	}
	cfg, _, err := loadConfig(repo)
	if err != nil {
		return nil, err
	}
	key := mustKey(top)
	file, err := a.Store.Read(ctx)
	if err != nil {
		return nil, err
	}
	if ws, ok := file.Workspaces[key]; ok {
		_ = a.touch(ctx, key)
		if setup {
			if err := a.prepare(ctx, cfg, ws); err != nil {
				return nil, err
			}
		}
		return a.view(ctx, ws, true)
	}
	branch, _ := gitx.CurrentBranch(ctx, top)
	slug := filepath.Base(top)
	if strings.HasPrefix(branch, worktree.BranchPrefix) {
		slug = strings.TrimPrefix(branch, worktree.BranchPrefix)
	}
	portNames := append([]string{}, cfg.Ports...)
	if runtime.GOOS == "windows" && len(cfg.Processes) > 0 {
		portNames = append(portNames, "pc")
	}
	allocated, err := a.allocate(ctx, portNames)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(config.DataDir(top), 0o755); err != nil {
		return nil, err
	}
	if setup {
		_ = worktree.ApplyInclude(ctx, repo, top)
		_ = copyfs.CopyDirs(ctx, repo, top, cfg.CopyDirs)
	}
	ws := state.Workspace{
		ID:         newID(),
		Slug:       slug,
		Path:       key,
		Repo:       mustKey(repo),
		Branch:     branch,
		Base:       cfg.Base,
		Ports:      allocated,
		CreatedAt:  time.Now().UTC(),
		LastUsedAt: time.Now().UTC(),
	}
	if err := a.Store.Update(ctx, func(f *state.File) error {
		f.Workspaces[ws.Path] = ws
		return nil
	}); err != nil {
		return nil, err
	}
	if setup {
		if err := a.prepare(ctx, cfg, ws); err != nil {
			return nil, err
		}
	} else {
		_ = a.writeEnv(cfg, ws)
	}
	return a.view(ctx, ws, true)
}

func (a *App) Attach(ctx context.Context, slug string) (*WorkspaceView, error) {
	ws, err := a.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	_ = a.touch(ctx, ws.Path)
	return a.view(ctx, ws, true)
}

func (a *App) Done(ctx context.Context, slug string, force bool) error {
	ws, err := a.resolve(ctx, slug)
	if err != nil {
		return err
	}
	if !force {
		dirty, extra, err := worktree.Dirty(ctx, ws.Path)
		if err == nil && dirty {
			return fmt.Errorf("worktree is dirty. Commit changes or use --force\n%s", extra)
		}
		if unpublished, why, err := worktree.Unpublished(ctx, ws.Path); err == nil && unpublished {
			return fmt.Errorf("branch %s is unpublished: %s. Push the branch or use --force", ws.Branch, why)
		}
	}
	cfg, _, _ := loadConfig(ws.Repo)
	_ = a.Down(ctx, ws.Path)
	if cfg != nil {
		env := workspaceEnv(cfg, ws)
		_ = runHooks(ctx, ws.Path, cfg.Hooks.Teardown, env)
	}
	if err := worktree.Remove(ctx, ws.Repo, ws.Path, force); err != nil {
		return err
	}
	_ = worktree.DeleteBranch(ctx, ws.Repo, ws.Branch)
	return a.Store.Update(ctx, func(f *state.File) error {
		delete(f.Workspaces, ws.Path)
		return nil
	})
}

func (a *App) List(ctx context.Context) ([]WorkspaceView, error) {
	file, err := a.Store.Read(ctx)
	if err != nil {
		return nil, err
	}
	var out []WorkspaceView
	for _, ws := range file.Workspaces {
		v, err := a.view(ctx, ws, false)
		if err != nil {
			continue
		}
		out = append(out, *v)
	}
	sort.Slice(out, func(i, j int) bool {
		if out[i].LastUsed.Equal(out[j].LastUsed) {
			return out[i].Slug < out[j].Slug
		}
		return out[i].LastUsed.After(out[j].LastUsed)
	})
	return out, nil
}

func (a *App) Status(ctx context.Context, slug string) (*WorkspaceView, error) {
	ws, err := a.resolve(ctx, slug)
	if err != nil {
		return nil, err
	}
	_ = a.touch(ctx, ws.Path)
	return a.view(ctx, ws, true)
}

func (a *App) Up(ctx context.Context, pathOrSlug string) error {
	ws, err := a.resolve(ctx, pathOrSlug)
	if err != nil {
		return err
	}
	cfg, _, err := loadConfig(ws.Repo)
	if err != nil {
		return err
	}
	if len(cfg.Processes) == 0 {
		return fmt.Errorf("no processes declared in lane.yaml. Add a processes: section or skip lane up")
	}
	env := workspaceEnv(cfg, ws)
	if err := process.Render(ws.Path, cfg.Processes, env); err != nil {
		return err
	}
	if err := a.writeEnv(cfg, ws); err != nil {
		return err
	}
	_ = a.touch(ctx, ws.Path)
	return process.Up(ctx, ws.Path, env, ws.Ports["pc"])
}

func (a *App) Down(ctx context.Context, pathOrSlug string) error {
	ws, err := a.resolve(ctx, pathOrSlug)
	if err != nil {
		here := cwd()
		if process.Running(ctx, here) {
			return process.Down(ctx, here)
		}
		if pathOrSlug != "" && process.Running(ctx, pathOrSlug) {
			return process.Down(ctx, pathOrSlug)
		}
		return err
	}
	return process.Down(ctx, ws.Path)
}

func (a *App) Logs(ctx context.Context, slug, name string) error {
	ws, err := a.resolve(ctx, slug)
	if err != nil {
		return err
	}
	if name == "" {
		return fmt.Errorf("process name required. Run lane status --json to list process names")
	}
	return process.Logs(ctx, ws.Path, name, os.Stdout, os.Stderr)
}

func (a *App) Run(ctx context.Context, slug string, argv []string) error {
	if len(argv) == 0 {
		return fmt.Errorf("missing command. Usage: lane run -- <cmd>")
	}
	ws, err := a.resolve(ctx, slug)
	if err != nil {
		return err
	}
	cfg, _, err := loadConfig(ws.Repo)
	if err != nil {
		return err
	}
	env := workspaceEnv(cfg, ws)
	_ = a.touch(ctx, ws.Path)
	cmd := exec.CommandContext(ctx, argv[0], argv[1:]...)
	cmd.Dir = ws.Path
	cmd.Env = config.Environ(os.Environ(), env)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func (a *App) Reset(ctx context.Context, slug string) error {
	ws, err := a.resolve(ctx, slug)
	if err != nil {
		return err
	}
	_ = a.Down(ctx, ws.Path)
	if err := os.RemoveAll(config.DataDir(ws.Path)); err != nil {
		return err
	}
	if err := os.MkdirAll(config.DataDir(ws.Path), 0o755); err != nil {
		return err
	}
	cfg, _, err := loadConfig(ws.Repo)
	if err != nil {
		return err
	}
	return a.prepare(ctx, cfg, ws)
}

func (a *App) Init(ctx context.Context, force bool) error {
	repo, err := gitx.MainRepo(ctx, cwd())
	if err != nil {
		return fmt.Errorf("not a git repository: %w", err)
	}
	path := filepath.Join(repo, config.Filename)
	if _, err := os.Stat(path); err == nil && !force {
		return fmt.Errorf("%s already exists. Edit it in place or pass --force to overwrite", path)
	}
	if err := os.WriteFile(path, []byte(config.Template), 0o644); err != nil {
		return err
	}
	if err := ensureGitignore(repo); err != nil {
		return err
	}
	return skill.Install(repo)
}

func (a *App) SkillInstall(_ context.Context) error {
	repo, err := gitx.MainRepo(context.Background(), cwd())
	if err != nil {
		return fmt.Errorf("not a git repository: %w", err)
	}
	return skill.Install(repo)
}

func (a *App) HookInstall(ctx context.Context, which string) error {
	repo, err := gitx.MainRepo(ctx, cwd())
	if err != nil {
		return fmt.Errorf("not a git repository: %w", err)
	}
	if which == "" {
		which = "all"
	}
	if err := skill.WriteHooks(repo); err != nil {
		return err
	}
	switch which {
	case "cursor", "all":
		if err := writeCursorHook(repo); err != nil {
			return err
		}
	}
	switch which {
	case "claude", "all":
		if err := writeClaudeHook(repo); err != nil {
			return err
		}
	case "cursor":
	default:
		if which != "all" {
			return fmt.Errorf("unknown harness %q. Use cursor, claude, or all", which)
		}
	}
	return nil
}

func (a *App) GC(ctx context.Context, dry bool) (*gc.Report, error) {
	return gc.Run(ctx, a.Store, gc.Options{
		DryRun: dry,
		Down:   process.Down,
		Remove: func(ctx context.Context, ws state.Workspace, force bool) error {
			return a.Done(ctx, ws.Path, true)
		},
	})
}

func (a *App) Doctor(ctx context.Context, fix bool) (*DoctorReport, error) {
	rep := &DoctorReport{Checks: []DoctorCheck{}}
	ok := DoctorCheck{Name: "git", OK: true}
	if err := gitx.Has(ctx); err != nil {
		ok.OK = false
		ok.Detail = err.Error()
	} else if v, err := gitx.Run(ctx, "", "version"); err == nil {
		ok.Detail = v
	}
	rep.Checks = append(rep.Checks, ok)

	pc := DoctorCheck{Name: "process-compose", OK: true}
	bin, err := process.LookPath()
	if err != nil {
		if fix {
			bin, err = process.Download(ctx)
		}
		if err != nil {
			pc.OK = false
			pc.Detail = err.Error()
		}
	}
	if err == nil {
		if ver, vErr := process.Version(ctx, bin); vErr == nil {
			pc.Detail = ver + " (" + bin + ")"
		} else {
			pc.Detail = bin
		}
	}
	rep.Checks = append(rep.Checks, pc)

	st := DoctorCheck{Name: "state", OK: true, Detail: home.StatePath()}
	if _, err := a.Store.Read(ctx); err != nil {
		st.OK = false
		st.Detail = err.Error()
	}
	rep.Checks = append(rep.Checks, st)

	if fix {
		if _, err := a.GC(ctx, false); err != nil {
			rep.Checks = append(rep.Checks, DoctorCheck{Name: "gc", OK: false, Detail: err.Error()})
		} else {
			rep.Checks = append(rep.Checks, DoctorCheck{Name: "gc", OK: true, Detail: "reclaimed vanished workspaces"})
		}
	}
	rep.OK = true
	for _, c := range rep.Checks {
		if !c.OK {
			rep.OK = false
		}
	}
	return rep, nil
}

type DoctorReport struct {
	OK     bool          `json:"ok"`
	Checks []DoctorCheck `json:"checks"`
}

type DoctorCheck struct {
	Name   string `json:"name"`
	OK     bool   `json:"ok"`
	Detail string `json:"detail,omitempty"`
}

func (a *App) Open(ctx context.Context, slug, name string) (string, error) {
	ws, err := a.resolve(ctx, slug)
	if err != nil {
		return "", err
	}
	port, err := pickPort(ws.Ports, name)
	if err != nil {
		return "", err
	}
	url := fmt.Sprintf("http://127.0.0.1:%d", port)
	if err := openURL(url); err != nil {
		return url, fmt.Errorf("open %s: %w. Open the URL manually", url, err)
	}
	return url, nil
}

func pickPort(ports map[string]int, name string) (int, error) {
	if name != "" {
		p, ok := ports[name]
		if !ok {
			return 0, fmt.Errorf("port %q is not allocated. Run lane ports --json", name)
		}
		return p, nil
	}
	for _, cand := range []string{"web", "frontend", "ui", "http", "api"} {
		if p, ok := ports[cand]; ok && cand != "pc" {
			return p, nil
		}
	}
	for n, p := range ports {
		if n == "pc" {
			continue
		}
		return p, nil
	}
	return 0, fmt.Errorf("no application ports allocated. Declare ports: in lane.yaml")
}

func openURL(url string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	default:
		cmd = exec.Command("xdg-open", url)
	}
	return cmd.Start()
}
