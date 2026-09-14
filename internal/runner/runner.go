// Package runner makes services, hooks and one-off commands share one runtime.
package runner

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/Mrjwj34/berth/internal/config"
	"github.com/Mrjwj34/berth/internal/process"
	"github.com/Mrjwj34/berth/internal/state"
)

type Session struct {
	Config    *config.Config
	Workspace state.Workspace
}
type Plan struct {
	GatewayPorts     map[string]int `json:"gateway_ports,omitempty"`
	Backend          string         `json:"backend"`
	Engine           string         `json:"engine,omitempty"`
	Image            string         `json:"image,omitempty"`
	WorkingDir       string         `json:"working_dir"`
	DataDir          string         `json:"data_dir"`
	HostPorts        map[string]int `json:"host_ports"`
	Listen           map[string]int `json:"listen,omitempty"`
	NetworkNamespace bool           `json:"network_namespace"`
	SecuritySandbox  bool           `json:"security_sandbox"`
}

func New(cfg *config.Config, ws state.Workspace) (*Session, error) {
	if cfg == nil {
		return nil, fmt.Errorf("workspace configuration missing")
	}
	if cfg.Runtime != ws.Runtime || !reflect.DeepEqual(cfg.Listen, ws.Listen) {
		return nil, fmt.Errorf("runtime/port contract changed; create a new workspace")
	}
	expected := len(cfg.Ports)
	if runtime.GOOS == "windows" && cfg.Runtime.Kind() == "native" && len(cfg.Processes) > 0 {
		expected++
	}
	if len(ws.Ports) != expected {
		return nil, fmt.Errorf("port contract changed; create a new workspace")
	}
	for _, name := range cfg.Ports {
		if ws.Ports[name] == 0 {
			return nil, fmt.Errorf("port %s is not allocated", name)
		}
	}
	return &Session{Config: cfg, Workspace: ws}, nil
}

// Existing controls stored runtime resources even when berth.yaml is broken.
func Existing(ws state.Workspace) *Session { return &Session{Workspace: ws} }
func (s *Session) Plan() Plan {
	w := s.Workspace
	p := Plan{Backend: w.Runtime.Kind(), WorkingDir: w.Path, DataDir: config.DataDir(w.Path), HostPorts: w.Ports, Listen: w.Listen}
	if p.Backend == "container" {
		p.Engine = w.Runtime.EngineName()
		p.Image = w.Runtime.Image
		p.WorkingDir = "/workspace"
		p.DataDir = "/workspace/.berth/data"
		p.NetworkNamespace = true
		p.GatewayPorts = s.gatewayPorts()
	}
	return p
}
func (s *Session) HostEnv() map[string]string { return s.env(false) }
func (s *Session) Env() map[string]string     { return s.env(s.Workspace.Runtime.Kind() == "container") }
func (s *Session) env(container bool) map[string]string {
	w := s.Workspace
	path, repo, ports := w.Path, w.Repo, w.Ports
	if container {
		path, repo, ports = "/workspace", "/workspace", w.Listen
	}
	vars := config.IdentityVars(path, w.Slug, repo, w.Branch, ports)
	for name, p := range w.Ports {
		vars[strings.Replace(config.PortEnvName(name), "BERTH_PORT_", "BERTH_HOST_PORT_", 1)] = strconv.Itoa(p)
	}
	if container {
		vars["BERTH_DATA_DIR"] = "/workspace/.berth/data"
		vars["GIT_DIR"] = "/berth/git/worktrees/" + filepath.Base(w.GitDir)
		vars["GIT_WORK_TREE"] = "/workspace"
		vars["HOME"] = "/tmp/berth-home"
		vars["XDG_CACHE_HOME"] = "/tmp/berth-home/.cache"
	}
	if s.Config != nil {
		return config.MergeEnv(vars, s.Config.Env)
	}
	return vars
}
func (s *Session) Prepare(ctx context.Context) error {
	if s.Workspace.Runtime.Kind() == "container" {
		return s.ensureContainer(ctx)
	}
	return nil
}
func (s *Session) Up(ctx context.Context) error {
	if s.Config == nil || len(s.Config.Processes) == 0 {
		return fmt.Errorf("no processes declared in berth.yaml")
	}
	if err := s.Prepare(ctx); err != nil {
		return err
	}
	if err := process.RenderAt(s.Workspace.Path, s.Plan().WorkingDir, s.Config.Processes, s.Env(), effectiveShutdownTimeout(s.Config, s.Workspace)); err != nil {
		return err
	}
	if s.Workspace.Runtime.Kind() == "native" {
		warnNativeWindowsCommands(s)
		return process.Up(ctx, s.Workspace.Path, s.Env(), s.Workspace.Ports["pc"])
	}
	pc, err := s.pcRunning(ctx)
	if err != nil {
		return err
	}
	if !pc {
		argv := []string{"process-compose", "up", "-D", "-t=false", "--disable-dotenv", "-U", "-u", containerSocket, "-f", "/workspace/.berth/pc.yaml", "-L", "/workspace/.berth/process-compose.log"}
		if out, err := s.containerExec(ctx, argv, false).CombinedOutput(); err != nil {
			return fmt.Errorf("runtime process-compose up: %w\n%s", err, out)
		}
	}
	child, cancel := context.WithTimeout(ctx, 60*time.Second)
	defer cancel()
	return process.WaitReady(child, s.Status)
}

// effectiveShutdownTimeout bounds graceful shutdown for the session's runtime.
// Native Windows keeps the supervisor default: process-compose stops a native
// Windows process with taskkill /T /F, which cannot be ignored, and its timed
// shutdown path waits on process exit in a way that breaks `process-compose down`
// on Windows. Containers run Linux and use the bound on every host.
func effectiveShutdownTimeout(cfg *config.Config, ws state.Workspace) int {
	shutdown := cfg.ShutdownTimeoutSeconds
	if shutdown <= 0 {
		shutdown = config.DefaultShutdownTimeoutSeconds
	}
	if runtime.GOOS == "windows" && ws.Runtime.Kind() == "native" {
		return 0
	}
	return shutdown
}

// warnNativeWindowsCommands explains a platform limitation before it turns into
// a confusing runtime failure. process-compose starts a native process by
// splitting the command string on Windows instead of running it through a
// shell, so quotes stay part of the argument and a value containing a space
// arrives split in two — the program then reports an error about half a path.
// The warning goes to stderr because the CLI keeps stdout machine-readable.
func warnNativeWindowsCommands(s *Session) {
	if runtime.GOOS != "windows" || s.Config == nil {
		return
	}
	problems := process.WindowsCommandProblems(s.Config.Processes, s.Env())
	if len(problems) == 0 {
		return
	}
	for _, problem := range problems {
		fmt.Fprintln(os.Stderr, "berth: warning: "+problem)
	}
	fmt.Fprintln(os.Stderr, "berth: warning: "+process.WindowsCommandRemedy)
}

func (s *Session) Running(ctx context.Context) (bool, error) {
	if s.Workspace.Runtime.Kind() == "native" {
		return process.IsRunning(ctx, s.Workspace.Path)
	}
	info, err := s.inspect(ctx)
	if err != nil {
		return false, err
	}
	return info != nil && info.State.Running, nil
}
func (s *Session) Status(ctx context.Context) ([]process.Proc, error) {
	if s.Workspace.Runtime.Kind() == "native" {
		return process.Status(ctx, s.Workspace.Path)
	}
	running, err := s.Running(ctx)
	if err != nil || !running {
		return nil, err
	}
	pc, err := s.pcRunning(ctx)
	if err != nil || !pc {
		return nil, err
	}
	child, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	out, err := s.containerExec(child, []string{"process-compose", "process", "list", "-o", "json", "-U", "-u", containerSocket}, false).Output()
	if err != nil {
		return nil, err
	}
	return process.DecodeStatus(out)
}
func (s *Session) Down(ctx context.Context) error {
	if s.Workspace.Runtime.Kind() == "native" {
		return process.Down(ctx, s.Workspace.Path)
	}
	info, err := s.inspect(ctx)
	if err != nil || info == nil {
		return err
	}
	if !info.State.Running {
		return nil
	}
	grace, cancel := context.WithTimeout(ctx, 12*time.Second)
	_, _ = s.containerExec(grace, []string{"process-compose", "down", "-U", "-u", containerSocket}, false).CombinedOutput()
	cancel()
	// Engine stop covers every descendant, including hooks or unmanaged children.
	if _, err := s.engineOutput(ctx, "stop", "--time", "10", info.ID); err != nil {
		return err
	}
	info, err = s.inspect(ctx)
	if err != nil {
		return err
	}
	if info != nil && info.State.Running {
		return fmt.Errorf("runtime is still running; refusing destructive cleanup")
	}
	return nil
}
func (s *Session) Destroy(ctx context.Context) error {
	if err := s.Down(ctx); err != nil {
		return err
	}
	if s.Workspace.Runtime.Kind() == "native" {
		return nil
	}
	info, err := s.inspect(ctx)
	if err != nil || info == nil {
		return err
	}
	_, err = s.engineOutput(ctx, "rm", info.ID)
	return err
}
func (s *Session) Run(ctx context.Context, argv []string, stdin io.Reader, stdout, stderr io.Writer) error {
	if len(argv) == 0 {
		return fmt.Errorf("missing command")
	}
	if err := s.Prepare(ctx); err != nil {
		return err
	}
	var cmd *exec.Cmd
	if s.Workspace.Runtime.Kind() == "container" {
		cmd = s.containerExec(ctx, argv, stdin != nil)
	} else {
		cmd = exec.CommandContext(ctx, argv[0], argv[1:]...)
		cmd.Dir = s.Workspace.Path
		cmd.Env = config.Environ(os.Environ(), s.Env())
		configureCancellation(cmd)
	}
	cmd.Stdin = stdin
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	if ctx.Err() != nil && s.Workspace.Runtime.Kind() == "container" {
		// Killing the engine CLI alone can leave docker-exec's target running.
		cleanup, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()
		if stopErr := s.Down(cleanup); stopErr != nil {
			return fmt.Errorf("cancelled command: %v; stop runtime: %w", err, stopErr)
		}
	}
	return err
}
func (s *Session) Hooks(ctx context.Context, commands []string, stdout, stderr io.Writer) error {
	for _, line := range commands {
		if strings.TrimSpace(line) == "" {
			continue
		}
		argv := []string{"sh", "-c", line}
		if s.Workspace.Runtime.Kind() == "native" && runtime.GOOS == "windows" {
			argv = []string{"cmd", "/C", line}
		}
		if err := s.Run(ctx, argv, nil, stdout, stderr); err != nil {
			return fmt.Errorf("hook failed: %w", err)
		}
	}
	return nil
}
func (s *Session) Logs(ctx context.Context, name string, stdout, stderr io.Writer) error {
	if s.Workspace.Runtime.Kind() == "native" {
		return process.Logs(ctx, s.Workspace.Path, name, stdout, stderr)
	}
	info, err := s.inspect(ctx)
	if err != nil {
		return err
	}
	if info == nil || !info.State.Running {
		return fmt.Errorf("workspace runtime is not running")
	}
	cmd := s.containerExec(ctx, []string{"process-compose", "process", "logs", name, "-U", "-u", containerSocket}, false)
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}
