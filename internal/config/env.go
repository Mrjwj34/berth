package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"unicode"
)

const (
	EnvBegin = "# BEGIN LANE"
	EnvEnd   = "# END LANE"
)

var (
	braceVar = regexp.MustCompile(`\$\{([A-Za-z_][A-Za-z0-9_]*)\}`)
	plainVar = regexp.MustCompile(`\$([A-Za-z_][A-Za-z0-9_]*)`)
)

func PortEnvName(portName string) string {
	var b strings.Builder
	b.WriteString("LANE_PORT_")
	for _, r := range portName {
		switch {
		case r == '-' || r == '.':
			b.WriteByte('_')
		default:
			b.WriteRune(unicode.ToUpper(r))
		}
	}
	return b.String()
}

func DataDir(worktree string) string {
	return filepath.Join(worktree, ".lane", "data")
}

func LaneDir(worktree string) string {
	return filepath.Join(worktree, ".lane")
}

// IdentityVars are always injected for a workspace.
func IdentityVars(worktree, slug, repo, branch string, ports map[string]int) map[string]string {
	vars := map[string]string{
		"LANE_DATA_DIR":  DataDir(worktree),
		"LANE_WORKSPACE": worktree,
		"LANE_SLUG":      slug,
		"LANE_ROOT":      repo,
		"LANE_BRANCH":    branch,
	}
	for name, port := range ports {
		vars[PortEnvName(name)] = fmt.Sprintf("%d", port)
	}
	return vars
}

func Expand(s string, vars map[string]string) string {
	s = braceVar.ReplaceAllStringFunc(s, func(m string) string {
		name := braceVar.FindStringSubmatch(m)[1]
		if v, ok := vars[name]; ok {
			return v
		}
		return m
	})
	s = plainVar.ReplaceAllStringFunc(s, func(m string) string {
		name := plainVar.FindStringSubmatch(m)[1]
		if v, ok := vars[name]; ok {
			return v
		}
		return m
	})
	return s
}

func ExpandMap(in map[string]string, vars map[string]string) map[string]string {
	out := make(map[string]string, len(in))
	for k, v := range in {
		out[k] = Expand(v, vars)
	}
	return out
}

func MergeEnv(identity map[string]string, declared map[string]string) map[string]string {
	merged := make(map[string]string, len(identity)+len(declared))
	for k, v := range identity {
		merged[k] = v
	}
	for k, v := range ExpandMap(declared, merged) {
		merged[k] = v
	}
	return merged
}

func Environ(base []string, vars map[string]string) []string {
	keys := make(map[string]int, len(base))
	out := append([]string(nil), base...)
	for i, kv := range out {
		if name, _, ok := strings.Cut(kv, "="); ok {
			keys[name] = i
		}
	}
	for k, v := range vars {
		entry := k + "=" + v
		if i, ok := keys[k]; ok {
			out[i] = entry
			continue
		}
		out = append(out, entry)
	}
	return out
}

func WriteEnvFile(path string, vars map[string]string) error {
	if path == "" {
		return nil
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	existing, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("read %s: %w", path, err)
	}
	body := stripManaged(string(existing))
	block := renderManaged(vars)
	var b strings.Builder
	b.WriteString(strings.TrimRight(body, "\n"))
	if b.Len() > 0 {
		b.WriteByte('\n')
	}
	b.WriteString(block)
	if err := os.WriteFile(path, []byte(b.String()), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", path, err)
	}
	return nil
}

func stripManaged(s string) string {
	start := strings.Index(s, EnvBegin)
	if start < 0 {
		return s
	}
	end := strings.Index(s[start:], EnvEnd)
	if end < 0 {
		return strings.TrimSpace(s[:start]) + "\n"
	}
	end = start + end + len(EnvEnd)
	if end < len(s) && s[end] == '\n' {
		end++
	}
	return s[:start] + s[end:]
}

func renderManaged(vars map[string]string) string {
	keys := make([]string, 0, len(vars))
	for k := range vars {
		keys = append(keys, k)
	}
	// stable order: LANE_* first then others, each group sorted
	sortStrings(keys)
	var b strings.Builder
	b.WriteString(EnvBegin)
	b.WriteByte('\n')
	for _, k := range keys {
		fmt.Fprintf(&b, "%s=%s\n", k, vars[k])
	}
	b.WriteString(EnvEnd)
	b.WriteByte('\n')
	return b.String()
}

func sortStrings(s []string) {
	for i := 1; i < len(s); i++ {
		for j := i; j > 0 && s[j] < s[j-1]; j-- {
			s[j], s[j-1] = s[j-1], s[j]
		}
	}
}
