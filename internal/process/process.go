package process

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/Mrjwj34/berth/internal/config"
	"github.com/Mrjwj34/berth/internal/home"
	"gopkg.in/yaml.v3"
)

type Proc struct {
	Name    string `json:"name"`
	Status  string `json:"status"`
	Ready   string `json:"ready,omitempty"`
	PID     int    `json:"pid,omitempty"`
	Exit    int    `json:"exit_code,omitempty"`
	Healthy bool   `json:"healthy"`
}

type pcProcess struct {
	Name          string `json:"name"`
	Status        string `json:"status"`
	IsReady       string `json:"is_ready"`
	HasReadyProbe bool   `json:"has_ready_probe"`
	PID           int    `json:"pid"`
	ExitCode      int    `json:"exit_code"`
	Namespace     string `json:"namespace"`
}

func PCFile(worktree string) string { return filepath.Join(config.BerthDir(worktree), "pc.yaml") }
func Socket(worktree string) string {
	old := filepath.Join(config.BerthDir(worktree), "pc.sock")
	if _, err := os.Lstat(old); err == nil {
		return old
	}
	abs, err := filepath.Abs(worktree)
	if err != nil {
		abs = worktree
	}
	sum := sha256.Sum256([]byte(filepath.Clean(abs)))
	base := filepath.Join(home.Dir(), "run")
	p := filepath.Join(base, hex.EncodeToString(sum[:8]), "pc.sock")
	if runtime.GOOS != "windows" && len(p) >= 100 {
		user := sha256.Sum256([]byte(home.Dir()))
		p = filepath.Join(os.TempDir(), "berth-"+hex.EncodeToString(user[:6]), hex.EncodeToString(sum[:8])+".sock")
	}
	return p
}
func LogFile(worktree string) string {
	return filepath.Join(config.BerthDir(worktree), "process-compose.log")
}
func PortFile(worktree string) string {
	old := filepath.Join(config.BerthDir(worktree), "pc.port")
	if _, err := os.Lstat(old); err == nil {
		return old
	}
	return filepath.Join(filepath.Dir(Socket(worktree)), "pc.port")
}

// RenderAt keeps host paths out of configurations executed inside a runtime.
// shutdownSeconds greater than zero bounds how long a process may ignore SIGTERM
// before process-compose kills its process group; zero leaves termination to the
// supervisor's default.
func RenderAt(worktree, workingDir string, processes map[string]any, env map[string]string, shutdownSeconds int) error {
	if len(processes) == 0 {
		return nil
	}
	expanded := expandAny(processes, env)
	procs, ok := expanded.(map[string]any)
	if !ok {
		return fmt.Errorf("processes must be a mapping")
	}
	envList := make([]any, 0, len(env))
	for k, v := range env {
		envList = append(envList, k+"="+v)
	}
	for name, raw := range procs {
		proc, ok := raw.(map[string]any)
		if !ok {
			return fmt.Errorf("processes.%s must be a mapping", name)
		}
		if _, exists := proc["working_dir"]; !exists {
			proc["working_dir"] = workingDir
		}
		existing, ok := proc["environment"].([]any)
		if proc["environment"] != nil && !ok {
			return fmt.Errorf("processes.%s.environment must be a list", name)
		}
		for _, item := range existing {
			text, ok := item.(string)
			if !ok {
				return fmt.Errorf("processes.%s.environment requires KEY=VALUE strings", name)
			}
			key, _, ok := strings.Cut(text, "=")
			if !ok {
				return fmt.Errorf("invalid environment entry %q", text)
			}
			if strings.HasPrefix(key, "BERTH_") || key == "GIT_DIR" || key == "GIT_WORK_TREE" {
				return fmt.Errorf("process cannot override runtime identity %s", key)
			}
		}
		proc["environment"] = append(append([]any{}, envList...), existing...)
		// Bound graceful shutdown so a process that ignores SIGTERM cannot make
		// process-compose down wait forever. A shutdown block declared by the
		// project is authoritative and never rewritten.
		if shutdownSeconds > 0 {
			if _, exists := proc["shutdown"]; !exists {
				proc["shutdown"] = map[string]any{"timeout_seconds": shutdownSeconds}
			}
		}
		procs[name] = proc
	}
	doc := map[string]any{
		"version":   "0.5",
		"processes": procs,
	}
	data, err := yaml.Marshal(doc)
	if err != nil {
		return fmt.Errorf("encode pc.yaml: %w", err)
	}
	if err := os.MkdirAll(config.BerthDir(worktree), 0o755); err != nil {
		return err
	}
	return os.WriteFile(PCFile(worktree), data, 0o600)
}

func expandAny(v any, env map[string]string) any {
	switch t := v.(type) {
	case string:
		return config.Expand(t, env)
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			out[k] = expandAny(val, env)
			if k == "port" {
				if text, ok := out[k].(string); ok {
					if n, err := strconv.Atoi(text); err == nil {
						out[k] = n
					}
				}
			}
		}
		return out
	case []any:
		out := make([]any, len(t))
		for i, val := range t {
			out[i] = expandAny(val, env)
		}
		return out
	default:
		return v
	}
}

func Up(ctx context.Context, worktree string, env map[string]string, pcPort int) error {
	if _, err := os.Stat(PCFile(worktree)); err != nil {
		return fmt.Errorf("no generated process file at %s. Add a processes: section to berth.yaml", PCFile(worktree))
	}
	running, err := IsRunning(ctx, worktree)
	if err != nil {
		return err
	}
	if running {
		return waitReady(ctx, worktree, 60*time.Second)
	}
	bin, err := Ensure(ctx)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(Socket(worktree)), 0o700); err != nil {
		return err
	}
	if err := ensureToken(worktree); err != nil {
		return err
	}
	args := []string{
		"up", "-D", "-t=false", "--address", "127.0.0.1",
		"-f", PCFile(worktree),
		"--disable-dotenv",
		"-L", LogFile(worktree),
	}
	control, err := clientArgs(worktree, pcPort)
	if err != nil {
		return err
	}
	args = append(args, control...)
	if runtime.GOOS == "windows" {
		args = append(args[:1], args[2:]...) // omit unsupported -D
		if pcPort <= 0 {
			return fmt.Errorf("missing process-compose control port")
		}
		log, err := os.OpenFile(LogFile(worktree), os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o600)
		if err != nil {
			return err
		}
		cmd := exec.Command(bin, args...)
		configureDetached(cmd)
		cmd.Dir = worktree
		cmd.Env = pcEnviron(config.Environ(os.Environ(), env))
		cmd.Stdout = log
		cmd.Stderr = log
		if err := cmd.Start(); err != nil {
			_ = log.Close()
			return err
		}
		// The recorded PID covers the window before the supervisor answers: a
		// concurrent status must not mistake a starting supervisor for a dead one.
		if err := writePCPID(worktree, cmd.Process.Pid); err != nil {
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			_ = log.Close()
			return err
		}
		if err := os.WriteFile(PortFile(worktree), []byte(strconv.Itoa(pcPort)), 0o600); err != nil {
			_ = os.Remove(PIDFile(worktree))
			_ = cmd.Process.Kill()
			_ = cmd.Wait()
			_ = log.Close()
			return err
		}
		go func() { _ = cmd.Wait(); _ = log.Close() }()
		// From the first answer on, the control endpoint is the witness.
		defer func() { _ = os.Remove(PIDFile(worktree)) }()
		return waitReady(ctx, worktree, 60*time.Second)
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = worktree
	cmd.Env = pcEnviron(config.Environ(os.Environ(), env))
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("process-compose up: %w\n%s\nInspect %s or run berth doctor", err, buf.String(), LogFile(worktree))
	}
	return waitReady(ctx, worktree, 60*time.Second)
}

func Down(ctx context.Context, worktree string) error {
	running, err := IsRunning(ctx, worktree)
	if err != nil {
		return err
	}
	if !running {
		return nil
	}
	before, err := queryStatus(ctx, worktree)
	if err != nil {
		return err
	}

	bin, err := LookPath()
	if err != nil {
		return err
	}
	child, cancel := context.WithTimeout(ctx, downTimeout(worktree))
	defer cancel()
	control, err := clientArgs(worktree, readPCPort(worktree))
	if err != nil {
		return err
	}
	args := append([]string{"down"}, control...)
	cmd := exec.CommandContext(child, bin, args...)
	cmd.Env = pcEnviron(os.Environ())
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("process-compose down: %w\n%s", err, cleanSupervisorOutput(out))
	}
	// Do not delete data until the previously observed managed processes exited.
	// PID reuse only causes conservative retention; no PID is killed here.
	for {
		alive := false
		for _, p := range before {
			if processAlive(p.PID) {
				alive = true
				break
			}
		}
		if !alive {
			break
		}
		select {
		case <-child.Done():
			return fmt.Errorf("processes remain after down; refusing cleanup: %w", child.Err())
		case <-time.After(50 * time.Millisecond):
		}
	}

	if err := os.Remove(Socket(worktree)); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.Remove(PortFile(worktree)); err != nil && !os.IsNotExist(err) {
		return err
	}
	if err := os.Remove(PIDFile(worktree)); err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

// downTimeout leaves room for process-compose to escalate to SIGKILL after the
// workspace's shutdown timeout, so the local client is not killed first.
func downTimeout(worktree string) time.Duration {
	grace := config.DefaultShutdownTimeoutSeconds
	if cfg, err := config.Load(filepath.Join(worktree, config.Filename)); err == nil && cfg.ShutdownTimeoutSeconds > 0 {
		grace = cfg.ShutdownTimeoutSeconds
	}
	d := time.Duration(grace)*time.Second + 20*time.Second
	if d < 30*time.Second {
		d = 30 * time.Second
	}
	return d
}

// cleanSupervisorOutput drops process-compose's debug chatter, which otherwise
// buries the actual failure (for example signal: killed).
func cleanSupervisorOutput(out []byte) string {
	var lines []string
	for _, line := range strings.Split(string(out), "\n") {
		text := strings.TrimSpace(line)
		if text == "" || strings.Contains(text, `"level":"debug"`) {
			continue
		}
		lines = append(lines, text)
	}
	return strings.Join(lines, "\n")
}

func Status(ctx context.Context, worktree string) ([]Proc, error) {
	running, err := IsRunning(ctx, worktree)
	if err != nil {
		return nil, err
	}
	if !running {
		return nil, nil
	}
	return queryStatus(ctx, worktree)
}
func queryStatus(ctx context.Context, worktree string) ([]Proc, error) {
	control, err := clientArgs(worktree, readPCPort(worktree))
	if err != nil {
		return nil, err
	}
	bin, err := LookPath()
	if err != nil {
		return nil, err
	}
	child, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	args := append([]string{"process", "list", "-o", "json"}, control...)
	cmd := exec.CommandContext(child, bin, args...)
	cmd.Env = pcEnviron(os.Environ())
	out, err := cmd.Output()
	if err != nil {
		return nil, err
	}
	return DecodeStatus(out)
}
func DecodeStatus(data []byte) ([]Proc, error) {
	var raw []pcProcess
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("decode process status: %w", err)
	}
	procs := make([]Proc, 0, len(raw))
	for _, p := range raw {
		// process-compose reports "-" both without probes and before probes finish.
		healthy := strings.EqualFold(p.Status, "Running") && (strings.EqualFold(p.IsReady, "Ready") || (!p.HasReadyProbe && p.IsReady == "-"))
		procs = append(procs, Proc{Name: p.Name, Status: p.Status, Ready: p.IsReady, PID: p.PID, Exit: p.ExitCode, Healthy: healthy})
	}
	return procs, nil
}

func Logs(ctx context.Context, worktree, name string, stdout, _ io.Writer) error {
	if err := validateToken(TokenFile(worktree)); err != nil {
		return err
	}
	token, err := os.ReadFile(TokenFile(worktree))
	if err != nil {
		return err
	}

	client := &http.Client{}
	address := "http://127.0.0.1:" + strconv.Itoa(readPCPort(worktree))
	if runtime.GOOS != "windows" {
		address = "http://localhost"
		client.Transport = &http.Transport{DialContext: func(ctx context.Context, _, _ string) (net.Conn, error) {
			return (&net.Dialer{}).DialContext(ctx, "unix", Socket(worktree))
		}}
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, address+"/process/logs/"+url.PathEscape(name)+"/0/0", nil)
	if err != nil {
		return err
	}
	req.Header.Set("X-PC-Token-Key", strings.TrimSpace(string(token)))
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("read process logs: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		return fmt.Errorf("read process logs: HTTP %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}
	var result struct {
		Logs []string `json:"logs"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return fmt.Errorf("decode process logs: %w", err)
	}
	for _, line := range result.Logs {
		if _, err := fmt.Fprintln(stdout, line); err != nil {
			return err
		}
	}
	return nil
}

// IsRunning distinguishes absence from an unresponsive supervisor. Destructive
// operations must preserve data when process state cannot be established. A
// supervisor that exited on its own is not such a state: once it is provably
// gone, its stale control files are reclaimed and the workspace counts as
// stopped, because only a conversation with the supervisor ever removed them.
func IsRunning(ctx context.Context, worktree string) (bool, error) {
	endpoint := Socket(worktree)
	if runtime.GOOS == "windows" {
		endpoint = PortFile(worktree)
	}
	if _, err := os.Lstat(endpoint); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	if _, err := queryStatus(ctx, worktree); err != nil {
		if !supervisorGone(ctx, worktree) {
			return false, fmt.Errorf("process state unknown: %s exists but the supervisor does not answer (inspect %s). Nothing destructive runs while the state is unknown; if no process-compose is running for this workspace, remove %s, %s and the socket beside them (on Windows they live under BERTH_HOME/run), then retry: %w", endpoint, LogFile(worktree), PortFile(worktree), TokenFile(worktree), err)
		}
		if rerr := reapStaleSupervisor(worktree); rerr != nil {
			return false, fmt.Errorf("supervisor exited but its control files at %s could not be reclaimed: %w", endpoint, rerr)
		}
		warnReclaimedSupervisor(worktree, endpoint)
		return false, nil
	}
	return true, nil
}

func TokenFile(worktree string) string {
	return strings.TrimSuffix(Socket(worktree), ".sock") + ".token"
}
func ensureToken(worktree string) error {
	path := TokenFile(worktree)
	if st, err := os.Lstat(path); err == nil {
		if !st.Mode().IsRegular() {
			return fmt.Errorf("invalid API token file")
		}
		return validateToken(path)
	} else if !os.IsNotExist(err) {
		return err
	}
	var token [32]byte
	if _, err := rand.Read(token[:]); err != nil {
		return err
	}
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_EXCL, 0o600)
	if err != nil {
		return err
	}
	_, err = f.WriteString(hex.EncodeToString(token[:]) + "\n")
	closeErr := f.Close()
	if err != nil {
		return err
	}
	return closeErr
}
func pcEnviron(base []string) []string {
	return config.Environ(base, map[string]string{"PC_API_TOKEN": "", "PC_API_TOKEN_PATH": ""})
}
func validateToken(path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read API token: %w", err)
	}
	token, err := hex.DecodeString(strings.TrimSpace(string(data)))
	if err != nil || len(token) != 32 {
		return fmt.Errorf("invalid API token in %s; verify the supervisor is stopped before removing this file", path)
	}
	return nil
}
func clientArgs(worktree string, pcPort int) ([]string, error) {
	var args []string
	if runtime.GOOS != "windows" {
		args = []string{"-U", "-u", Socket(worktree)}
	} else {
		if pcPort == 0 {
			pcPort = readPCPort(worktree)
		}
		if pcPort < 1 || pcPort > 65535 {
			return nil, fmt.Errorf("invalid or missing supervisor control port in %s", PortFile(worktree))
		}
		args = []string{"--address", "127.0.0.1", "-p", strconv.Itoa(pcPort)}
	}
	if err := validateToken(TokenFile(worktree)); err != nil {
		return nil, err
	}
	return append(args, "--token-file", TokenFile(worktree)), nil
}

func readPCPort(worktree string) int {
	data, err := os.ReadFile(PortFile(worktree))
	if err != nil {
		return 0
	}
	n, _ := strconv.Atoi(strings.TrimSpace(string(data)))
	return n
}

func waitReady(ctx context.Context, worktree string, timeout time.Duration) error {
	child, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()
	return WaitReady(child, func(ctx context.Context) ([]Proc, error) { return queryStatus(ctx, worktree) })
}
func WaitReady(ctx context.Context, status func(context.Context) ([]Proc, error)) error {
	var last []Proc
	var lastErr error
	for {
		last, lastErr = status(ctx)
		if lastErr == nil && allReady(last) {
			return nil
		}
		for _, p := range last {
			if strings.EqualFold(p.Status, "Error") || (strings.EqualFold(p.Status, "Completed") && p.Exit != 0) {
				return fmt.Errorf("process %s failed with exit %d (%s)", p.Name, p.Exit, p.Status)
			}
		}
		select {
		case <-ctx.Done():
			return fmt.Errorf("processes not ready: %w; status=%+v; last error=%v", ctx.Err(), last, lastErr)
		case <-time.After(100 * time.Millisecond):
		}
	}
}
func allReady(procs []Proc) bool {
	if len(procs) == 0 {
		return false
	}
	for _, p := range procs {
		if strings.EqualFold(p.Status, "Completed") && p.Exit == 0 {
			continue
		}
		if strings.EqualFold(p.Status, "Skipped") {
			continue
		}
		if !p.Healthy {
			return false
		}
	}
	return true
}
