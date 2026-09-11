package remap

import (
	"os"
	"runtime"
	"strings"

	"github.com/Mrjwj34/lane/internal/config"
)

const (
	envFile = "LANE_REMAP_FILE"
	envOn   = "LANE_REMAP"
	envLib  = "LANE_REMAP_LIB"
)

// EnvOn is set on remapped process environments.
const EnvOn = envOn

// EnvVars are injected into process-compose and `lane run` so bind/connect remap.
func EnvVars(worktree string) map[string]string {
	lib := LibPath()
	out := map[string]string{
		envOn:   "1",
		envFile: TablePath(worktree),
		envLib:  LibPath(),
	}
	switch runtime.GOOS {
	case "linux":
		out["LD_PRELOAD"] = joinPreload(os.Getenv("LD_PRELOAD"), lib)
	case "darwin":
		out["DYLD_INSERT_LIBRARIES"] = joinPreload(os.Getenv("DYLD_INSERT_LIBRARIES"), lib)
	}
	return out
}

// ApplyMap copies env and adds remap variables.
func ApplyMap(env map[string]string, worktree string) map[string]string {
	out := make(map[string]string, len(env)+4)
	for k, v := range env {
		out[k] = v
	}
	for k, v := range EnvVars(worktree) {
		if k == "LD_PRELOAD" || k == "DYLD_INSERT_LIBRARIES" {
			out[k] = joinPreload(out[k], v)
			continue
		}
		out[k] = v
	}
	return out
}

// Environ appends remap variables onto a process environment list.
func Environ(base []string, worktree string) []string {
	return config.Environ(base, EnvVars(worktree))
}

// WithoutPreload drops interceptor libraries so process-compose itself is
// not rewritten. Child processes still receive EnvVars via pc.yaml.
func WithoutPreload(env map[string]string) map[string]string {
	if env == nil {
		return nil
	}
	out := make(map[string]string, len(env))
	for k, v := range env {
		if k == "LD_PRELOAD" || k == "DYLD_INSERT_LIBRARIES" {
			continue
		}
		out[k] = v
	}
	return out
}

func joinPreload(existing, lib string) string {
	if lib == "" {
		return existing
	}
	if existing == "" {
		return lib
	}
	parts := strings.Split(existing, string(os.PathListSeparator))
	for _, p := range parts {
		if p == lib {
			return existing
		}
	}
	return lib + string(os.PathListSeparator) + existing
}
