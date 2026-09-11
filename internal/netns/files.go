package netns

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/Mrjwj34/lane/internal/config"
)

const (
	roleEnv = "LANE_INTERNAL_NETNS"
	specEnv = "LANE_INTERNAL_NETNS_SPEC"
)

// Mapping publishes a unique host port to the TCP port a process already binds.
type Mapping struct {
	Name   string `json:"name"`
	Host   int    `json:"host"`
	Listen int    `json:"listen"`
}

type spec struct {
	Worktree string    `json:"worktree"`
	Maps     []Mapping `json:"maps"`
	Bin      string    `json:"bin"`
	Args     []string  `json:"args"`
	Env      []string  `json:"env"`
}

func specPath(worktree string) string {
	return filepath.Join(config.LaneDir(worktree), "isolate.json")
}

func supervisePIDPath(worktree string) string {
	return filepath.Join(config.LaneDir(worktree), "isolate.pid")
}

func insidePIDPath(worktree string) string {
	return filepath.Join(config.LaneDir(worktree), "netns.pid")
}

func isolateLogPath(worktree string) string {
	return filepath.Join(config.LaneDir(worktree), "isolate.log")
}

func fwdDir(worktree string) string {
	return filepath.Join(config.LaneDir(worktree), "fwd")
}

func fwdSock(worktree, name string) string {
	return filepath.Join(fwdDir(worktree), name+".sock")
}

func writeSpec(worktree string, s spec) error {
	if err := os.MkdirAll(config.LaneDir(worktree), 0o755); err != nil {
		return err
	}
	data, err := json.MarshalIndent(s, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(specPath(worktree), data, 0o644)
}

func loadSpec() (spec, error) {
	path := os.Getenv(specEnv)
	data, err := os.ReadFile(path)
	if err != nil {
		return spec{}, err
	}
	var s spec
	if err := json.Unmarshal(data, &s); err != nil {
		return spec{}, err
	}
	return s, nil
}

func writePID(path string, pid int) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(strconv.Itoa(pid)+"\n"), 0o644)
}

func readPID(path string) int {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0
	}
	n, _ := strconv.Atoi(strings.TrimSpace(string(data)))
	return n
}

// InsidePID is the process that holds the workspace network namespace.
func InsidePID(worktree string) int {
	return readPID(insidePIDPath(worktree))
}

func supervisePID(worktree string) int {
	return readPID(supervisePIDPath(worktree))
}

func alive(pid int) bool {
	if pid <= 0 {
		return false
	}
	proc, err := os.FindProcess(pid)
	if err != nil {
		return false
	}
	return processAlive(proc)
}
