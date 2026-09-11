package app

import (
	"bytes"
	"encoding/json"
	"github.com/Mrjwj34/lane/internal/skill"
	"os"
	"path/filepath"
	"strings"
)

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
