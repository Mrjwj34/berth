package app

import (
	"context"
	"fmt"
	"github.com/Mrjwj34/berth/internal/config"
	"github.com/Mrjwj34/berth/internal/gitx"
	"github.com/Mrjwj34/berth/internal/home"
	"github.com/Mrjwj34/berth/internal/process"
	"github.com/Mrjwj34/berth/internal/skill"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"
)

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

// InstallResult reports one artifact an install command wrote, or found
// already installed.
type InstallResult struct {
	Agent string
	Scope string
	// Path is relative to the scope root: the repository for project scope,
	// the user profile for user scope.
	Path    string
	Changed bool
}

// SkillInstall writes the embedded skill into every selected harness
// directory. A bare `berth skill install` keeps writing only the shared
// .agents/skills/berth copy; --agent, --all and --scope widen that.
func (a *App) SkillInstall(ctx context.Context, opts skill.Options) ([]InstallResult, error) {
	opts, err := opts.Normalize()
	if err != nil {
		return nil, err
	}
	repo, err := gitx.MainRepo(ctx, cwd())
	if err != nil {
		if opts.Scope != skill.ScopeUser {
			return nil, fmt.Errorf("not a git repository: %w", err)
		}
		// User scope writes into the profile only, so it works outside a
		// repository; no project target is produced without one.
		repo = ""
	}
	home, err := skill.HomeDir()
	if err != nil {
		return nil, err
	}
	targets, err := opts.SkillTargets(repo, home)
	if err != nil {
		return nil, err
	}
	out := make([]InstallResult, 0, len(targets))
	for _, target := range targets {
		changed, err := skill.InstallInto(target.Dir)
		if err != nil {
			return out, err
		}
		out = append(out, InstallResult{Agent: target.Harness.Name, Scope: target.Scope, Path: target.Rel(), Changed: changed})
	}
	return out, nil
}

// HookInstall merges berth's worktree adapter for the selected harnesses into
// that harness's own configuration file. Hooks are project-scoped: a worktree
// is created inside a repository, so there is no user-scope hook to write.
func (a *App) HookInstall(ctx context.Context, opts skill.Options) ([]InstallResult, error) {
	opts, err := opts.Normalize()
	if err != nil {
		return nil, err
	}
	if opts.Scope == skill.ScopeUser {
		return nil, nil
	}
	repo, err := gitx.MainRepo(ctx, cwd())
	if err != nil {
		return nil, fmt.Errorf("not a git repository: %w", err)
	}
	hooks, err := opts.HookTargets()
	if err != nil {
		return nil, err
	}
	out := make([]InstallResult, 0, len(hooks))
	for _, h := range hooks {
		var changed bool
		switch h.Name {
		case "cursor":
			changed, err = writeCursorHook(repo)
		case "windsurf":
			changed, err = writeWindsurfHook(repo)
		case "claude":
			changed, err = writeClaudeHook(repo)
		default:
			err = fmt.Errorf("harness %q has no worktree hook berth installs", h.Name)
		}
		if err != nil {
			return out, err
		}
		out = append(out, InstallResult{Agent: h.Name, Scope: skill.ScopeProject, Path: h.HookFile, Changed: changed})
	}
	return out, nil
}

// Agents reports which harnesses berth supports and what is already installed
// in this repository and in the user profile. It only reads.
func (a *App) Agents(ctx context.Context) (*skill.Report, error) {
	repo, err := gitx.MainRepo(ctx, cwd())
	if err != nil {
		return nil, fmt.Errorf("not a git repository: %w", err)
	}
	home, err := skill.HomeDir()
	if err != nil {
		return nil, err
	}
	rep := skill.BuildReport(repo, home)
	return &rep, nil
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

	cfg := config.Defaults()
	if top, err := gitx.TopLevel(ctx, cwd()); err == nil {
		loaded, _, err := loadConfig(top)
		if err != nil {
			rep.Checks = append(rep.Checks, DoctorCheck{Name: "configuration", OK: false, Detail: err.Error()})
		} else {
			cfg = *loaded
		}
	}
	if cfg.Runtime.Kind() == "container" {
		child, cancel := context.WithTimeout(ctx, 10*time.Second)
		defer cancel()
		out, err := exec.CommandContext(child, cfg.Runtime.EngineName(), "info").CombinedOutput()
		check := DoctorCheck{Name: "container-engine", OK: err == nil, Detail: cfg.Runtime.EngineName()}
		if err != nil {
			check.Detail = fmt.Sprintf("%v: %s", err, out)
		}
		rep.Checks = append(rep.Checks, check)
		out, err = exec.CommandContext(child, cfg.Runtime.EngineName(), "image", "inspect", "--format", "{{.Os}}", cfg.Runtime.Image).CombinedOutput()
		check = DoctorCheck{Name: "runtime-image", OK: err == nil && strings.TrimSpace(string(out)) == "linux", Detail: cfg.Runtime.Image}
		if !check.OK {
			check.Detail = fmt.Sprintf("prepare Linux image %s once: %v %s", cfg.Runtime.Image, err, out)
		}
		rep.Checks = append(rep.Checks, check)
	} else if len(cfg.Processes) > 0 || fix {
		check := DoctorCheck{Name: "process-compose", OK: true}
		bin, err := process.LookPath()
		if err != nil && fix {
			bin, err = process.Download(ctx)
		}
		if err != nil {
			check.OK = false
			check.Detail = err.Error()
		} else {
			check.Detail = bin
		}
		rep.Checks = append(rep.Checks, check)
	}

	st := DoctorCheck{Name: "state", OK: true, Detail: home.StatePath()}
	if _, err := a.Store.Read(ctx); err != nil {
		st.OK = false
		st.Detail = err.Error()
	}
	rep.Checks = append(rep.Checks, st)

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
			return 0, fmt.Errorf("port %q is not allocated. Run berth ports --json", name)
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
	return 0, fmt.Errorf("no application ports allocated. Declare ports: in berth.yaml")
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
	if err := cmd.Start(); err != nil {
		return err
	}
	go func() { _ = cmd.Wait() }()
	return nil
}
