package skill

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestInstallWritesSkillCopies(t *testing.T) {
	root := t.TempDir()
	if err := Install(root); err != nil {
		t.Fatal(err)
	}
	for _, rel := range []string{
		".agents/skills/lane/SKILL.md",
		".claude/skills/lane/SKILL.md",
		".cursor/skills/lane/SKILL.md",
	} {
		data, err := os.ReadFile(filepath.Join(root, rel))
		if err != nil {
			t.Fatal(err)
		}
		if len(data) < 100 {
			t.Fatalf("%s too small", rel)
		}
	}
}

func TestWriteHooks(t *testing.T) {
	root := t.TempDir()
	if err := WriteHooks(root); err != nil {
		t.Fatal(err)
	}
	p := filepath.Join(root, ".lane", "hooks", "claude-worktree-create.sh")
	st, err := os.Stat(p)
	if err != nil {
		t.Fatal(err)
	}
	if runtime.GOOS != "windows" && st.Mode()&0o111 == 0 {
		t.Fatalf("hook is not executable: %s", st.Mode())
	}
}
