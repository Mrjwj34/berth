package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const Filename = "lane.yaml"

var ErrNotFound = errors.New("lane.yaml not found")

type Config struct {
	Version      int               `yaml:"version" json:"version"`
	Base         string            `yaml:"base" json:"base"`
	WorktreeRoot string            `yaml:"worktree_root,omitempty" json:"worktree_root,omitempty"`
	Ports        Ports             `yaml:"ports" json:"ports"`
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
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse %s: %w", path, err)
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

func (c *Config) validate() error {
	if c.Version != 1 {
		return fmt.Errorf("unsupported version %d (supported: 1)", c.Version)
	}
	seen := map[string]struct{}{}
	for _, port := range c.Ports {
		if port.Name == "" {
			return fmt.Errorf("ports: empty name")
		}
		if strings.ContainsAny(port.Name, " \t") {
			return fmt.Errorf("ports: %q contains whitespace", port.Name)
		}
		if port.Listen < 0 || port.Listen > 65535 {
			return fmt.Errorf("ports.%s: listen %d is not a valid TCP port", port.Name, port.Listen)
		}
		key := strings.ToLower(port.Name)
		if _, ok := seen[key]; ok {
			return fmt.Errorf("ports: duplicate name %q", port.Name)
		}
		seen[key] = struct{}{}
	}
	return nil
}

// Find walks from startDir toward the filesystem root looking for lane.yaml.
// Missing file is not an error: L0 repos work with defaults.
func Find(startDir string) (cfg *Config, root string, err error) {
	dir, err := filepath.Abs(startDir)
	if err != nil {
		return nil, "", err
	}
	for {
		path := filepath.Join(dir, Filename)
		if st, err := os.Stat(path); err == nil && !st.IsDir() {
			cfg, err := Load(path)
			if err != nil {
				return nil, "", err
			}
			return cfg, dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			d := Defaults()
			return &d, startDir, nil
		}
		dir = parent
	}
}

func Exists(root string) bool {
	st, err := os.Stat(filepath.Join(root, Filename))
	return err == nil && !st.IsDir()
}
