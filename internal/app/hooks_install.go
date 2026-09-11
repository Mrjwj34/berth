package app

import (
	"encoding/json"
	"github.com/Mrjwj34/berth/internal/skill"
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
	if strings.Contains(s, ".berth/") {
		return nil
	}
	var b strings.Builder
	b.WriteString(s)
	if s != "" && !strings.HasSuffix(s, "\n") {
		b.WriteByte('\n')
	}
	b.WriteString("\n# berth workspaces\n.berth/\n")
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

// writeCursorHook merges the Cursor adapter into .cursor/worktrees.json. Cursor
// runs it inside every worktree it creates, so those worktrees get registered
// with berth instead of being recreated by it.
func writeCursorHook(repo string) error {
	data, err := skill.Adapter("cursor.worktrees.json")
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
