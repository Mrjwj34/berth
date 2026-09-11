package app

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/Mrjwj34/lane/internal/config"
	"github.com/Mrjwj34/lane/internal/gitx"
	"github.com/Mrjwj34/lane/internal/ports"
	"github.com/Mrjwj34/lane/internal/process"
	"github.com/Mrjwj34/lane/internal/remap"
	"github.com/Mrjwj34/lane/internal/skill"
	"github.com/Mrjwj34/lane/internal/state"
	"github.com/Mrjwj34/lane/internal/worktree"
)

func cwd() string {
	dir, err := os.Getwd()
	if err != nil {
		return "."
	}
	return dir
}

func loadConfig(repo string) (*config.Config, string, error) {
	if config.Exists(repo) {
		cfg, err := config.Load(filepath.Join(repo, config.Filename))
		return cfg, repo, err
	}
	d := config.Defaults()
	return &d, repo, nil
}

func (a *App) lookupSlug(ctx context.Context, repo, slug string) (state.Workspace, bool) {
	file, err := a.Store.Read(ctx)
	if err != nil {
		return state.Workspace{}, false
	}
	return file.BySlug(repo, slug)
}

func (a *App) resolve(ctx context.Context, slugOrPath string) (state.Workspace, error) {
	file, err := a.Store.Read(ctx)
	if err != nil {
		return state.Workspace{}, err
	}
	if slugOrPath != "" {
		if ws, ok := file.Lookup(slugOrPath); ok {
			return ws, nil
		}
		if repo, err := gitx.MainRepo(ctx, cwd()); err == nil {
			if ws, ok := file.BySlug(repo, slugOrPath); ok {
				return ws, nil
			}
		}
		for _, ws := range file.Workspaces {
			if ws.Slug == slugOrPath || ws.Path == slugOrPath {
				return ws, nil
			}
		}
		return state.Workspace{}, fmt.Errorf("workspace %q not found. Run lane ls", slugOrPath)
	}
	here := mustKey(cwd())
	for path, ws := range file.Workspaces {
		if here == path || strings.HasPrefix(here, path+string(filepath.Separator)) {
			return ws, nil
		}
	}
	if top, err := gitx.TopLevel(ctx, cwd()); err == nil {
		if ws, ok := file.Lookup(top); ok {
			return ws, nil
		}
	}
	return state.Workspace{}, fmt.Errorf("no workspace for this directory. Run lane new <slug> or lane adopt")
}

func (a *App) allocate(ctx context.Context, names []string) (map[string]int, error) {
	if len(names) == 0 {
		return map[string]int{}, nil
	}
	file, err := a.Store.Read(ctx)
	if err != nil {
		return nil, err
	}
	return ports.Allocate(ctx, file.UsedPorts(), names)
}

func (a *App) touch(ctx context.Context, path string) error {
	return a.Store.Update(ctx, func(f *state.File) error {
		ws, ok := f.Workspaces[path]
		if !ok {
			return nil
		}
		state.Touch(&ws)
		f.Workspaces[path] = ws
		return nil
	})
}

func (a *App) prepare(ctx context.Context, cfg *config.Config, ws state.Workspace) error {
	if err := os.MkdirAll(config.DataDir(ws.Path), 0o755); err != nil {
		return err
	}
	if err := a.writeEnv(cfg, ws); err != nil {
		return err
	}
	env := workspaceEnv(cfg, ws)
	return runHooks(ctx, ws.Path, cfg.Hooks.Setup, env)
}

func (a *App) writeEnv(cfg *config.Config, ws state.Workspace) error {
	env := workspaceEnv(cfg, ws)
	if cfg.EnvFile == "" {
		return nil
	}
	path := cfg.EnvFile
	if !filepath.IsAbs(path) {
		path = filepath.Join(ws.Path, path)
	}
	return config.WriteEnvFile(path, env)
}

func usedPortList(ctx context.Context, a *App) []int {
	file, err := a.Store.Read(ctx)
	if err != nil {
		return nil
	}
	out := make([]int, 0, len(file.UsedPorts()))
	for p := range file.UsedPorts() {
		out = append(out, p)
	}
	return out
}

func (a *App) publishDiscovered(ctx context.Context, cfg *config.Config, ws state.Workspace) error {
	named := map[int]struct{}{}
	for _, p := range cfg.Ports {
		if p.Listen > 0 {
			named[p.Listen] = struct{}{}
		}
	}
	for name := range ws.Ports {
		if n, err := strconv.Atoi(name); err == nil {
			named[n] = struct{}{}
		}
	}
	deadline := time.Now().Add(3 * time.Second)
	var extra []remap.Mapping
	for {
		extra = extra[:0]
		for _, m := range remap.Mappings(ws.Path) {
			if m.Listen <= 0 || m.Host <= 0 {
				continue
			}
			if _, ok := named[m.Listen]; ok {
				continue
			}
			extra = append(extra, m)
		}
		if len(extra) > 0 || time.Now().After(deadline) {
			break
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(80 * time.Millisecond):
		}
	}
	if len(extra) == 0 {
		return nil
	}
	return a.Store.Update(ctx, func(f *state.File) error {
		cur, ok := f.Workspaces[ws.Path]
		if !ok {
			return nil
		}
		if cur.Ports == nil {
			cur.Ports = map[string]int{}
		}
		for _, m := range extra {
			cur.Ports[strconv.Itoa(m.Listen)] = m.Host
		}
		f.Workspaces[ws.Path] = cur
		return nil
	})
}

func mergeListen(declared map[string]int, ports map[string]int) map[string]int {
	out := map[string]int{}
	for name, listen := range declared {
		out[name] = listen
	}
	for name := range ports {
		if out[name] > 0 {
			continue
		}
		n, err := strconv.Atoi(name)
		if err != nil || n <= 0 || n > 65535 {
			continue
		}
		out[name] = n
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

func workspaceEnv(cfg *config.Config, ws state.Workspace) map[string]string {
	id := config.IdentityVars(ws.Path, ws.Slug, ws.Repo, ws.Branch, ws.Ports)
	for _, p := range cfg.Ports {
		if p.Listen > 0 {
			id[config.ListenEnvName(p.Name)] = strconv.Itoa(p.Listen)
		}
	}
	return config.MergeEnv(id, cfg.Env)
}

func (a *App) view(ctx context.Context, ws state.Workspace, withEnv bool) (*WorkspaceView, error) {
	cfg, _, _ := loadConfig(ws.Repo)
	if cfg == nil {
		d := config.Defaults()
		cfg = &d
	}
	dirty, _, _ := worktree.Dirty(ctx, ws.Path)
	v := &WorkspaceView{
		Slug:      ws.Slug,
		Path:      ws.Path,
		Repo:      ws.Repo,
		Branch:    ws.Branch,
		Ports:     ws.Ports,
		Listen:    mergeListen(cfg.Ports.ListenMap(), ws.Ports),
		Running:   process.Running(ctx, ws.Path),
		Dirty:     dirty,
		CreatedAt: ws.CreatedAt,
		LastUsed:  ws.LastUsedAt,
	}
	if withEnv {
		v.Env = workspaceEnv(cfg, ws)
	}
	if v.Running {
		if procs, err := process.Status(ctx, ws.Path); err == nil {
			v.Processes = procs
		}
	}
	return v, nil
}

func runHooks(ctx context.Context, dir string, commands []string, env map[string]string) error {
	for _, line := range commands {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var cmd *exec.Cmd
		if runtime.GOOS == "windows" {
			cmd = exec.CommandContext(ctx, "cmd", "/C", line)
		} else {
			cmd = exec.CommandContext(ctx, "sh", "-c", line)
		}
		cmd.Dir = dir
		cmd.Env = config.Environ(os.Environ(), env)
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return fmt.Errorf("hook %q failed: %w. Fix the command or run lane doctor", line, err)
		}
	}
	return nil
}

func mustKey(path string) string {
	k, err := state.Key(path)
	if err != nil {
		return filepath.Clean(path)
	}
	return k
}

func newID() string {
	var b [8]byte
	if _, err := rand.Read(b[:]); err != nil {
		return fmt.Sprintf("%d", os.Getpid())
	}
	return fmt.Sprintf("%x", b[:])
}

func ensureGitignore(repo string) error {
	path := filepath.Join(repo, ".gitignore")
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	s := string(data)
	if strings.Contains(s, ".lane/") {
		return nil
	}
	var b strings.Builder
	b.WriteString(s)
	if s != "" && !strings.HasSuffix(s, "\n") {
		b.WriteByte('\n')
	}
	b.WriteString("\n# lane workspaces\n.lane/\n")
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

func writeCursorHook(repo string) error {
	data, err := skill.HookAsset("cursor.worktrees.json")
	if err != nil {
		return err
	}
	dest := filepath.Join(repo, ".cursor", "worktrees.json")
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	if existing, err := os.ReadFile(dest); err == nil {
		var cur map[string]any
		var add map[string]any
		if json.Unmarshal(existing, &cur) == nil && json.Unmarshal(data, &add) == nil {
			for k, v := range add {
				cur[k] = v
			}
			if merged, err := json.MarshalIndent(cur, "", "  "); err == nil {
				return os.WriteFile(dest, append(merged, '\n'), 0o644)
			}
		}
	}
	return os.WriteFile(dest, data, 0o644)
}

func writeClaudeHook(repo string) error {
	data, err := skill.HookAsset("claude.settings.json")
	if err != nil {
		return err
	}
	dest := filepath.Join(repo, ".claude", "settings.json")
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	if existing, err := os.ReadFile(dest); err == nil && len(bytes.TrimSpace(existing)) > 0 {
		var cur map[string]any
		var add map[string]any
		if json.Unmarshal(existing, &cur) == nil && json.Unmarshal(data, &add) == nil {
			mergeMaps(cur, add)
			if merged, err := json.MarshalIndent(cur, "", "  "); err == nil {
				return os.WriteFile(dest, append(merged, '\n'), 0o644)
			}
		}
	}
	return os.WriteFile(dest, data, 0o644)
}

func mergeMaps(dst, src map[string]any) {
	for k, v := range src {
		if existing, ok := dst[k].(map[string]any); ok {
			if incoming, ok := v.(map[string]any); ok {
				mergeMaps(existing, incoming)
				continue
			}
		}
		dst[k] = v
	}
}
