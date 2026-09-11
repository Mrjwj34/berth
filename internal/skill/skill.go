// Package skill embeds the agent-facing assets berth ships: the SKILL.md that
// teaches a coding agent how to drive berth, the reference files that body points
// at, and the per-harness adapter files that wire a harness's own worktree
// lifecycle into berth.
//
// The skill is installed to the shared .agents/skills/berth/ convention by
// default, which twelve harnesses read (see harnesses.go). Claude Code and Cline
// do not read that convention, so they get their own copy under .claude/skills/
// and .cline/skills/ when they are named explicitly. Every location holds
// SKILL.md plus a references/ directory the body reaches by pointer. An adapter
// file has to stay where its harness looks for it, so adapters live beside the
// harness's own configuration and only reference berth commands.
package skill

import (
	"bytes"
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
)

//go:embed SKILL.md references/* adapters/*
var assets embed.FS

// AgentsDir is the directory the shared .agents/skills convention lives in.
const AgentsDir = ".agents"

// SkillPath is where the embedded SKILL.md is installed in the shared
// convention. Harnesses that read .agents/skills discover it directly.
func SkillPath(repoRoot string) string {
	return filepath.Join(repoRoot, AgentsDir, "skills", "berth", "SKILL.md")
}

func Content() ([]byte, error) {
	return assets.ReadFile("SKILL.md")
}

// Adapter returns the per-harness adapter fragment for name.
func Adapter(name string) ([]byte, error) {
	return assets.ReadFile(filepath.ToSlash(filepath.Join("adapters", name)))
}

// Install writes the whole embedded skill into the shared
// .agents/skills/berth/ location of repoRoot.
func Install(repoRoot string) error {
	_, err := InstallInto(filepath.Dir(SkillPath(repoRoot)))
	return err
}

// InstallInto writes the whole embedded skill into dir: the body and every
// reference it points at, so a pointer in the body resolves on disk. Adapters
// are not part of the skill and are written by the hook installer instead.
//
// It reports whether anything changed, so a repeated install is a no-op that
// leaves identical files byte-identical, and it writes through a temporary file
// and a rename: no harness documents a partial-write contract, so a reader must
// never observe half a skill.
func InstallInto(dir string) (bool, error) {
	files, err := skillFiles()
	if err != nil {
		return false, err
	}
	changed := false
	for _, name := range files {
		data, err := assets.ReadFile(name)
		if err != nil {
			return changed, fmt.Errorf("embed %s: %w", name, err)
		}
		dest := filepath.Join(dir, filepath.FromSlash(name))
		same, err := sameContent(dest, data)
		if err != nil {
			return changed, err
		}
		if same {
			continue
		}
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return changed, fmt.Errorf("create %s: %w", filepath.Dir(dest), err)
		}
		if err := writeFileAtomic(dest, data, 0o644); err != nil {
			return changed, fmt.Errorf("write %s: %w", dest, err)
		}
		changed = true
	}
	return changed, nil
}

// sameContent reports whether path already holds exactly data. A read error
// other than "does not exist" is returned rather than treated as a change, so a
// permission problem cannot silently look like a successful rewrite.
func sameContent(path string, data []byte) (bool, error) {
	existing, err := os.ReadFile(path)
	if err == nil {
		return bytes.Equal(existing, data), nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, fmt.Errorf("read %s: %w", path, err)
}

func writeFileAtomic(path string, data []byte, perm os.FileMode) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(name)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(name)
		return err
	}
	if err := os.Chmod(name, perm); err != nil {
		_ = os.Remove(name)
		return err
	}
	if err := os.Rename(name, path); err != nil {
		_ = os.Remove(name)
		return err
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
