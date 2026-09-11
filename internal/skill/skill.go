// Package skill embeds the agent-facing assets lane ships: the SKILL.md that
// teaches a coding agent how to drive lane, and the per-harness adapter files
// that wire a harness's own worktree lifecycle into lane.
//
// The skill is installed to one location, .agents/skills/lane/SKILL.md, which
// every supported harness reads (Cursor, Codex, pi, Antigravity). An adapter
// file has to stay where its harness looks for it, so adapters live beside the
// harness's own configuration and only reference lane commands.
package skill

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

//go:embed SKILL.md adapters/*
var assets embed.FS

// AgentsDir is the single directory lane installs its skill into.
const AgentsDir = ".agents"

// SkillPath is the one location the embedded SKILL.md is installed to. Harnesses
// that read .agents/skills discover it directly, so lane never writes a second
// copy for a harness that reads its own directory.
func SkillPath(repoRoot string) string {
	return filepath.Join(repoRoot, AgentsDir, "skills", "lane", "SKILL.md")
}

func Content() ([]byte, error) {
	return assets.ReadFile("SKILL.md")
}

// Adapter returns the per-harness adapter fragment for name.
func Adapter(name string) ([]byte, error) {
	return assets.ReadFile(filepath.ToSlash(filepath.Join("adapters", name)))
}

// Install writes the embedded SKILL.md into the single .agents skill directory.
func Install(repoRoot string) error {
	data, err := Content()
	if err != nil {
		return fmt.Errorf("embed SKILL.md: %w", err)
	}
	dest := SkillPath(repoRoot)
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(dest, data, 0o644); err != nil {
		return fmt.Errorf("write %s: %w", dest, err)
	}
	return nil
}
