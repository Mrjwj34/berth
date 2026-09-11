package runner

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

const containerSocket = "/tmp/lane-pc.sock"

var validID = regexp.MustCompile(`^[a-f0-9]{16,64}$`)

type containerInfo struct {
	ID    string `json:"Id"`
	Name  string `json:"Name"`
	State struct {
		Running bool `json:"Running"`
	} `json:"State"`
	Config struct {
		Labels map[string]string `json:"Labels"`
	} `json:"Config"`
}

func (s *Session) name() string { return "lane-" + s.Workspace.ID }
func digest(text string) string {
	sum := sha256.Sum256([]byte(text))
	return hex.EncodeToString(sum[:])
}
func (s *Session) specHash() string {
	w := s.Workspace
	data, _ := json.Marshal([]any{w.Runtime, w.Ports, w.Listen, w.GitDir, w.Path})
	return digest(string(data))
}
func (s *Session) engineOutput(ctx context.Context, args ...string) ([]byte, error) {
	child, cancel := context.WithTimeout(ctx, 30*time.Second)
	defer cancel()
	out, err := exec.CommandContext(child, s.Workspace.Runtime.EngineName(), args...).CombinedOutput()
	if err != nil {
		return nil, fmt.Errorf("%s %s: %w\n%s", s.Workspace.Runtime.EngineName(), strings.Join(args, " "), err, out)
	}
	return out, nil
}
func (s *Session) inspect(ctx context.Context) (*containerInfo, error) {
	if !validID.MatchString(s.Workspace.ID) {
		return nil, fmt.Errorf("invalid runtime identity")
	}
	// A daemon error is not evidence of absence. Never release resources on error.
	out, err := s.engineOutput(ctx, "container", "ls", "-a", "-q", "--filter", "name="+s.name())
	if err != nil {
		return nil, err
	}
	for _, id := range strings.Fields(string(out)) {
		raw, err := s.engineOutput(ctx, "container", "inspect", id)
		if err != nil {
			return nil, err
		}
		var items []containerInfo
		if err := json.Unmarshal(raw, &items); err != nil {
			return nil, err
		}
		if len(items) != 1 {
			return nil, fmt.Errorf("unexpected container inspect response")
		}
		info := items[0]
		if strings.TrimPrefix(info.Name, "/") != s.name() {
			continue
		}
		labels := info.Config.Labels
		if labels["dev.lane.workspace"] != s.Workspace.ID || labels["dev.lane.path"] != digest(s.Workspace.Path) || labels["dev.lane.spec"] != s.specHash() {
			return nil, fmt.Errorf("runtime ownership/configuration mismatch; refusing to control %s", s.name())
		}
		return &info, nil
	}
	return nil, nil
}
func (s *Session) createArgs() ([]string, error) {
	w := s.Workspace
	if !validID.MatchString(w.ID) {
		return nil, fmt.Errorf("invalid runtime identity")
	}
	if filepath.Base(filepath.Dir(w.GitDir)) != "worktrees" {
		return nil, fmt.Errorf("container backend requires linked-worktree Git metadata")
	}
	common := filepath.Dir(filepath.Dir(w.GitDir))
	for _, p := range []string{w.Path, common} {
		if strings.ContainsAny(p, ",\r\n") {
			return nil, fmt.Errorf("unsupported container mount path: %s", p)
		}
	}
	args := []string{"create", "--name", s.name(), "--pull=never", "--init", "--restart=no", "--label", "dev.lane.workspace=" + w.ID, "--label", "dev.lane.path=" + digest(w.Path), "--label", "dev.lane.spec=" + s.specHash(), "--security-opt", "no-new-privileges:true", "--cap-drop", "ALL", "--workdir", "/workspace", "--mount", "type=bind,src=" + w.Path + ",dst=/workspace", "--mount", "type=bind,src=" + common + ",dst=/lane/git", "--entrypoint", "bash"}
	user := w.Runtime.User
	if user == "" && runtime.GOOS == "linux" {
		user = strconv.Itoa(os.Getuid()) + ":" + strconv.Itoa(os.Getgid())
	}
	if w.Runtime.EngineName() == "podman" && runtime.GOOS == "linux" && os.Getuid() != 0 {
		args = append(args, "--userns=keep-id")
	}
	if user != "" {
		args = append(args, "--user", user)
	}
	if w.Runtime.Memory != "" {
		args = append(args, "--memory", w.Runtime.Memory)
	}
	if w.Runtime.CPUs > 0 {
		args = append(args, "--cpus", strconv.FormatFloat(w.Runtime.CPUs, 'f', -1, 64))
	}
	names := make([]string, 0, len(w.Ports))
	for name := range w.Ports {
		names = append(names, name)
	}
	sort.Strings(names)
	gateways := s.gatewayPorts()
	// Internal forwarding supports applications binding only to 127.0.0.1.
	// Never rewrite socket syscalls; reserve gateway ports explicitly in the plan.
	script := "set -euo pipefail\ncommand -v process-compose >/dev/null\ncommand -v socat >/dev/null\nmkdir -p /tmp/lane-home\nrm -f /tmp/lane-pc.sock\n"
	for _, name := range names {
		if w.Listen[name] <= 0 {
			return nil, fmt.Errorf("missing listen port for %s", name)
		}
		gateway := gateways[name]
		if gateway == 0 {
			return nil, fmt.Errorf("too many published ports")
		}

		args = append(args, "--publish", fmt.Sprintf("127.0.0.1:%d:%d/tcp", w.Ports[name], gateway))
		script += fmt.Sprintf("socat TCP-LISTEN:%d,bind=0.0.0.0,reuseaddr,fork TCP:127.0.0.1:%d &\n", gateway, w.Listen[name])
	}
	// Any gateway failure tears down PID 1; container lifecycle remains observable.
	script += "sleep infinity &\nwait -n\nexit 1\n"
	return append(args, w.Runtime.Image, "-c", script), nil
}
func (s *Session) ensureContainer(ctx context.Context) error {
	info, err := s.inspect(ctx)
	if err != nil {
		return err
	}
	if info == nil {
		out, err := s.engineOutput(ctx, "image", "inspect", "--format", "{{.Os}}", s.Workspace.Runtime.Image)
		if err != nil {
			return fmt.Errorf("prepare the image once before starting workspaces; no automatic pull or host fallback: %w", err)
		}
		if strings.TrimSpace(string(out)) != "linux" {
			return fmt.Errorf("runtime image must be Linux")
		}
		args, err := s.createArgs()
		if err != nil {
			return err
		}
		if _, err := s.engineOutput(ctx, args...); err != nil {
			return err
		}
		info, err = s.inspect(ctx)
		if err != nil {
			return err
		}
		if info == nil {
			return fmt.Errorf("created runtime disappeared")
		}
	}
	if !info.State.Running {
		if _, err := s.engineOutput(ctx, "start", info.ID); err != nil {
			return err
		}
	}
	child, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()
	out, err := s.containerExec(child, []string{"bash", "-c", "command -v process-compose && command -v socat && command -v git && test -w /workspace && mkdir -p /tmp/lane-home && git config --global --replace-all safe.directory /workspace"}, false).CombinedOutput()
	if err != nil {
		return fmt.Errorf("runtime preflight failed: %w\n%s", err, out)
	}
	return nil
}
func (s *Session) containerExec(ctx context.Context, argv []string, interactive bool) *exec.Cmd {
	args := []string{"exec", "--workdir", "/workspace"}
	if interactive {
		args = append(args, "-i")
	}
	env := s.Env()
	keys := make([]string, 0, len(env))
	for k := range env {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	for _, k := range keys {
		args = append(args, "--env", k+"="+env[k])
	}
	args = append(args, s.name())
	args = append(args, argv...)
	return exec.CommandContext(ctx, s.Workspace.Runtime.EngineName(), args...)
}
func (s *Session) pcRunning(ctx context.Context) (bool, error) {
	child, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	out, err := s.containerExec(child, []string{"sh", "-c", "if test -S /tmp/lane-pc.sock; then printf yes; else printf no; fi"}, false).Output()
	if err != nil {
		return false, err
	}
	return string(out) == "yes", nil
}

func (s *Session) gatewayPorts() map[string]int {
	used := map[int]bool{}
	for _, p := range s.Workspace.Listen {
		used[p] = true
	}
	names := make([]string, 0, len(s.Workspace.Ports))
	for name := range s.Workspace.Ports {
		names = append(names, name)
	}
	sort.Strings(names)
	out := map[string]int{}
	p := 65535
	for _, name := range names {
		for p > 1024 && used[p] {
			p--
		}
		if p <= 1024 {
			return nil
		}
		out[name] = p
		used[p] = true
		p--
	}
	return out
}
