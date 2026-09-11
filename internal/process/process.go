package process

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/Mrjwj34/lane/internal/config"
	"github.com/Mrjwj34/lane/internal/netns"
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
	Name      string `json:"name"`
	Status    string `json:"status"`
	IsReady   string `json:"is_ready"`
	PID       int    `json:"pid"`
	ExitCode  int    `json:"exit_code"`
	Namespace string `json:"namespace"`
}

func PCFile(worktree string) string { return filepath.Join(config.LaneDir(worktree), "pc.yaml") }
func Socket(worktree string) string { return filepath.Join(config.LaneDir(worktree), "pc.sock") }
func LogFile(worktree string) string {
	return filepath.Join(config.LaneDir(worktree), "process-compose.log")
}
func PortFile(worktree string) string { return filepath.Join(config.LaneDir(worktree), "pc.port") }

func Render(worktree string, processes map[string]any, env map[string]string) error {
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
			proc["working_dir"] = worktree
		}
		existing, _ := proc["environment"].([]any)
		proc["environment"] = append(append([]any{}, envList...), existing...)
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
	if err := os.MkdirAll(config.LaneDir(worktree), 0o755); err != nil {
		return err
	}
	return os.WriteFile(PCFile(worktree), data, 0o644)
}

func expandAny(v any, env map[string]string) any {
	switch t := v.(type) {
	case string:
		s := config.Expand(t, env)
		if n, err := strconv.Atoi(s); err == nil {
			return n
		}
		return s
	case map[string]any:
		out := make(map[string]any, len(t))
		for k, val := range t {
			out[k] = expandAny(val, env)
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

func Up(ctx context.Context, worktree string, env map[string]string, pcPort int, maps []netns.Mapping, isolate bool) error {
	if _, err := os.Stat(PCFile(worktree)); err != nil {
		return fmt.Errorf("no generated process file at %s. Add a processes: section to lane.yaml", PCFile(worktree))
	}
	if Running(ctx, worktree) {
		return nil
	}
	_ = netns.Stop(worktree)
	bin, err := Ensure(ctx)
	if err != nil {
		return err
	}
	args := []string{
		"up", "-t=false",
		"-f", PCFile(worktree),
		"--disable-dotenv",
		"-L", LogFile(worktree),
	}
	if !isolate {
		args = []string{
			"up", "-D", "-t=false",
			"-f", PCFile(worktree),
			"--disable-dotenv",
			"-L", LogFile(worktree),
		}
	}
	args = append(args, clientArgs(worktree, pcPort)...)
	if isolate {
		if err := netns.Start(ctx, worktree, maps, true, bin, args, config.Environ(os.Environ(), env)); err != nil {
			return err
		}
		return waitReady(ctx, worktree, 60*time.Second)
	}
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = worktree
	cmd.Env = config.Environ(os.Environ(), env)
	var buf bytes.Buffer
	cmd.Stdout = &buf
	cmd.Stderr = &buf
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("process-compose up: %w\n%s\nInspect %s or run lane doctor", err, buf.String(), LogFile(worktree))
	}
	if runtime.GOOS == "windows" && pcPort > 0 {
		_ = os.WriteFile(PortFile(worktree), []byte(strconv.Itoa(pcPort)), 0o644)
	}
	return waitReady(ctx, worktree, 60*time.Second)
}

func Down(ctx context.Context, worktree string) error {
	var downErr error
	if Running(ctx, worktree) {
		bin, err := LookPath()
		if err != nil {
			downErr = err
		} else {
			args := append([]string{"down"}, clientArgs(worktree, readPCPort(worktree))...)
			cmd := exec.CommandContext(ctx, bin, args...)
			cmd.Dir = worktree
			out, err := cmd.CombinedOutput()
			if err != nil {
				downErr = fmt.Errorf("process-compose down: %w\n%s", err, out)
			}
		}
	}
	_ = os.Remove(Socket(worktree))
	_ = os.Remove(PortFile(worktree))
	if err := netns.Stop(worktree); err != nil && downErr == nil {
		return err
	}
	return downErr
}

func Status(ctx context.Context, worktree string) ([]Proc, error) {
	if !Running(ctx, worktree) {
		return nil, nil
	}
	bin, err := LookPath()
	if err != nil {
		return nil, err
	}
	args := append([]string{"process", "list", "-o", "json"}, clientArgs(worktree, readPCPort(worktree))...)
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = worktree
	out, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("process-compose process list: %w", err)
	}
	var raw []pcProcess
	if err := json.Unmarshal(out, &raw); err != nil {
		return nil, fmt.Errorf("parse process list: %w", err)
	}
	procs := make([]Proc, 0, len(raw))
	for _, p := range raw {
		ready := p.IsReady
		healthy := strings.EqualFold(ready, "Ready") || strings.EqualFold(ready, "N/A") ||
			(strings.EqualFold(p.Status, "Running") && (ready == "" || strings.EqualFold(ready, "Unknown")))
		procs = append(procs, Proc{
			Name:    p.Name,
			Status:  p.Status,
			Ready:   ready,
			PID:     p.PID,
			Exit:    p.ExitCode,
			Healthy: healthy,
		})
	}
	return procs, nil
}

func Logs(ctx context.Context, worktree, name string, stdout, stderr io.Writer) error {
	bin, err := LookPath()
	if err != nil {
		return err
	}
	args := append([]string{"process", "logs", name}, clientArgs(worktree, readPCPort(worktree))...)
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = worktree
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	return cmd.Run()
}

func Running(ctx context.Context, worktree string) bool {
	if runtime.GOOS != "windows" {
		if st, err := os.Stat(Socket(worktree)); err == nil && st.Mode()&os.ModeSocket != 0 {
			return ping(ctx, worktree)
		}
		// some filesystems report sockets without ModeSocket
		if _, err := os.Stat(Socket(worktree)); err == nil {
			return ping(ctx, worktree)
		}
		return false
	}
	if readPCPort(worktree) == 0 {
		return false
	}
	return ping(ctx, worktree)
}

func ping(ctx context.Context, worktree string) bool {
	bin, err := LookPath()
	if err != nil {
		return false
	}
	args := append([]string{"process", "list", "-o", "json"}, clientArgs(worktree, readPCPort(worktree))...)
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = worktree
	return cmd.Run() == nil
}

func clientArgs(worktree string, pcPort int) []string {
	if runtime.GOOS != "windows" {
		return []string{"-U", "-u", Socket(worktree)}
	}
	if pcPort == 0 {
		pcPort = readPCPort(worktree)
	}
	if pcPort == 0 {
		return nil
	}
	return []string{"-p", strconv.Itoa(pcPort)}
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
	deadline := time.Now().Add(timeout)
	var last []Proc
	for time.Now().Before(deadline) {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !Running(ctx, worktree) {
			time.Sleep(150 * time.Millisecond)
			continue
		}
		procs, err := Status(ctx, worktree)
		if err != nil {
			time.Sleep(150 * time.Millisecond)
			continue
		}
		last = procs
		if len(procs) == 0 || allReady(procs) {
			return nil
		}
		time.Sleep(200 * time.Millisecond)
	}
	return fmt.Errorf("processes did not become ready within %s: %+v. Run lane logs <proc> or lane doctor", timeout, last)
}

func allReady(procs []Proc) bool {
	if len(procs) == 0 {
		return false
	}
	for _, p := range procs {
		if strings.EqualFold(p.Status, "Completed") || strings.EqualFold(p.Status, "Skipped") {
			continue
		}
		if strings.EqualFold(p.Ready, "Not Ready") {
			return false
		}
		if p.Healthy || strings.EqualFold(p.Status, "Running") {
			continue
		}
		return false
	}
	return true
}
