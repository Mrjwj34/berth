// Package skill embeds the agent-facing assets lane ships: the SKILL.md that
// teaches a coding agent how to drive lane, the reference files that body points
// at, and the per-harness adapter files that wire a harness's own worktree
// lifecycle into lane.
//
// The skill is installed to one location, .agents/skills/lane/, which every
// supported harness reads (Cursor, Codex, pi, Antigravity): SKILL.md plus a
// references/ directory the body reaches by pointer. An adapter file has to stay
// where its harness looks for it, so adapters live beside the harness's own
// configuration and only reference lane commands.
package skill

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

//go:embed SKILL.md references/* adapters/*
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

// Install writes the whole embedded skill into .agents/skills/lane: the body and
// every reference it points at, so a pointer in the body resolves on disk.
// Adapters are not part of the skill and are written by HookInstall instead.
func Install(repoRoot string) error {
	root := filepath.Dir(SkillPath(repoRoot))
	files, err := skillFiles()
	if err != nil {
		return err
	}
	for _, name := range files {
		data, err := assets.ReadFile(name)
		if err != nil {
			return fmt.Errorf("embed %s: %w", name, err)
		}
		dest := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dest, data, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", dest, err)
		}
	}
	return nil
}

// skillFiles lists the embedded files that belong to the skill itself: the body
// first, then its references in a stable order.
func skillFiles() ([]string, error) {
	refs, err := fs.Glob(assets, "references/*")
	if err != nil {
		return nil, err
	}
	sort.Strings(refs)
	return append([]string{"SKILL.md"}, refs...), nil
}
