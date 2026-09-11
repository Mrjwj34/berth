package config

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"math"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"gopkg.in/yaml.v3"
)

const Filename = "lane.yaml"

var ErrNotFound = errors.New("lane.yaml not found")

type Config struct {
	Version      int               `yaml:"version" json:"version"`
	Base         string            `yaml:"base" json:"base"`
	WorktreeRoot string            `yaml:"worktree_root,omitempty" json:"worktree_root,omitempty"`
	Runtime      Runtime           `yaml:"runtime,omitempty" json:"runtime,omitempty"`
	Listen       map[string]int    `yaml:"listen,omitempty" json:"listen,omitempty"`
	Ports        []string          `yaml:"ports" json:"ports"`
	Env          map[string]string `yaml:"env" json:"env,omitempty"`
	EnvFile      string            `yaml:"env_file,omitempty" json:"env_file,omitempty"`
	CopyDirs     []string          `yaml:"copy_dirs,omitempty" json:"copy_dirs,omitempty"`
	Hooks        Hooks             `yaml:"hooks" json:"hooks"`
	Processes    map[string]any    `yaml:"processes" json:"processes,omitempty"`
	GC           GC                `yaml:"gc" json:"gc"`
}

type Hooks struct {
	Setup    []string `yaml:"setup" json:"setup,omitempty"`
	Teardown []string `yaml:"teardown" json:"teardown,omitempty"`
}

type GC struct {
	IdleStopHours   int `yaml:"idle_stop_hours" json:"idle_stop_hours"`
	RemoveAfterDays int `yaml:"remove_after_days" json:"remove_after_days"`
	MaxWorkspaces   int `yaml:"max_workspaces" json:"max_workspaces"`
}

func Defaults() Config {
	return Config{
		Version:   1,
		Base:      "main",
		Env:       map[string]string{},
		Processes: map[string]any{},
	}
}

func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	cfg := Defaults()
	dec := yaml.NewDecoder(bytes.NewReader(data))
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
	}
	var extra any
	if err := dec.Decode(&extra); !errors.Is(err, io.EOF) {
		return nil, fmt.Errorf("%s: expected one YAML document", path)
	}
	if cfg.Version == 0 {
		cfg.Version = 1
	}
	if cfg.Base == "" {
		cfg.Base = "main"
	}
	if cfg.Env == nil {
		cfg.Env = map[string]string{}
	}
	if cfg.Processes == nil {
		cfg.Processes = map[string]any{}
	}
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	return &cfg, nil
}

type Runtime struct {
	Backend string  `yaml:"backend,omitempty" json:"backend,omitempty"`
	Engine  string  `yaml:"engine,omitempty" json:"engine,omitempty"`
	Image   string  `yaml:"image,omitempty" json:"image,omitempty"`
	User    string  `yaml:"user,omitempty" json:"user,omitempty"`
	Memory  string  `yaml:"memory,omitempty" json:"memory,omitempty"`
	CPUs    float64 `yaml:"cpus,omitempty" json:"cpus,omitempty"`
}

func (r Runtime) Kind() string {
	if r.Backend == "" {
		return "native"
	}
	return r.Backend
}
func (r Runtime) EngineName() string {
	if r.Engine == "" {
		return "docker"
	}
	return r.Engine
}

var validName = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9_.-]*$`)
var validEnv = regexp.MustCompile(`^[A-Za-z_][A-Za-z0-9_]*$`)
var validMemory = regexp.MustCompile(`^[1-9][0-9]*[bBkKmMgG]?$`)

func (c *Config) validate() error {
	if c.Version != 1 {
		return fmt.Errorf("unsupported version %d (supported: 1)", c.Version)
	}
	names, seen := map[string]bool{}, map[string]bool{}
	for _, name := range c.Ports {
		key := PortEnvName(name)
		if !validName.MatchString(name) || strings.EqualFold(name, "pc") {
			return fmt.Errorf("ports: invalid or reserved name %q", name)
		}
		if seen[key] {
			return fmt.Errorf("ports: duplicate environment name %s", key)
		}
		seen[key], names[name] = true, true
	}
	for name, port := range c.Listen {
		if !names[name] || port < 1 || port > 65535 {
			return fmt.Errorf("listen.%s must name a declared port and be in 1..65535", name)
		}
	}
	switch c.Runtime.Kind() {
	case "native":
		if len(c.Listen) > 0 || c.Runtime.Image != "" || c.Runtime.Engine != "" || c.Runtime.User != "" || c.Runtime.Memory != "" || c.Runtime.CPUs != 0 {
			return fmt.Errorf("listen and container options require runtime.backend: container")
		}
	case "container":
		if c.Runtime.Image == "" || strings.HasPrefix(c.Runtime.Image, "-") || strings.ContainsAny(c.Runtime.Image, " \t\r\n") {
			return fmt.Errorf("runtime.image must name a prebuilt Linux development image")
		}
		if e := c.Runtime.EngineName(); e != "docker" && e != "podman" {
			return fmt.Errorf("runtime.engine must be docker or podman")
		}
		for name := range names {
			if c.Listen[name] == 0 {
				return fmt.Errorf("container mode requires listen.%s", name)
			}
		}
		if c.Runtime.CPUs < 0 || math.IsNaN(c.Runtime.CPUs) || math.IsInf(c.Runtime.CPUs, 0) {
			return fmt.Errorf("runtime.cpus must be finite and nonnegative")
		}
		if c.Runtime.Memory != "" && !validMemory.MatchString(c.Runtime.Memory) {
			return fmt.Errorf("runtime.memory must be a positive size, e.g. 2g")
		}
	default:
		return fmt.Errorf("unsupported backend %q; use native or container", c.Runtime.Backend)
	}
	for name, value := range c.Env {
		if !validEnv.MatchString(name) || strings.HasPrefix(strings.ToUpper(name), "LANE_") || name == "GIT_DIR" || name == "GIT_WORK_TREE" {
			return fmt.Errorf("invalid or reserved environment key %q", name)
		}
		if strings.ContainsAny(value, "\x00\r\n") {
			return fmt.Errorf("env.%s: multiline values are not supported", name)
		}
	}
	if c.EnvFile != "" {
		if err := RelativePath(c.EnvFile); err != nil {
			return fmt.Errorf("env_file: %w", err)
		}
	}
	for _, p := range c.CopyDirs {
		if err := RelativePath(p); err != nil {
			return fmt.Errorf("copy_dirs: %w", err)
		}
	}
	if c.GC.IdleStopHours < 0 || c.GC.RemoveAfterDays < 0 || c.GC.MaxWorkspaces < 0 {
		return fmt.Errorf("gc values must be nonnegative")
	}
	return nil
}

// RelativePath rejects traversal and managed metadata, including Windows path
// syntax when validating configurations on Unix.
func RelativePath(p string) error {
	c := filepath.Clean(p)
	if c == "." || filepath.IsAbs(p) || filepath.VolumeName(p) != "" || strings.ContainsAny(p, "\\:\x00") || c == ".." || strings.HasPrefix(c, ".."+string(filepath.Separator)) {
		return fmt.Errorf("%q must be a repository-relative path", p)
	}
	for _, protected := range []string{".git", ".lane"} {
		if c == protected || strings.HasPrefix(c, protected+string(filepath.Separator)) {
			return fmt.Errorf("%q refers to managed metadata", p)
		}
	}
	return nil
}
