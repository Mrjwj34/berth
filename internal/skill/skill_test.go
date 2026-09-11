package skill

import (
	"os"
	"path/filepath"
	"testing"
)

func TestInstallWritesSkillUnderAgents(t *testing.T) {
	root := t.TempDir()
	if err := Install(root); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(SkillPath(root))
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 100 {
		t.Fatalf("%s too small", SkillPath(root))
	}
	for _, stale := range []string{
		".claude/skills/lane/SKILL.md",
		".cursor/skills/lane/SKILL.md",
		".lane/hooks",
	} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(stale))); !os.IsNotExist(err) {
			t.Fatalf("lane must not install outside .agents: %s", stale)
		}
	}
}
