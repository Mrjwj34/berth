package copyfs

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestCopyDirsHardlinkOrCopy(t *testing.T) {
	srcRoot := t.TempDir()
	dstRoot := t.TempDir()
	mod := filepath.Join(srcRoot, "node_modules", "pkg")
	if err := os.MkdirAll(mod, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(mod, "index.js"), []byte("ok"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CopyDirs(context.Background(), srcRoot, dstRoot, []string{"node_modules"}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(filepath.Join(dstRoot, "node_modules", "pkg", "index.js"))
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "ok" {
		t.Fatalf("got %q", got)
	}
}

func TestCopyDirsRejectsEscape(t *testing.T) {
	err := CopyDirs(context.Background(), t.TempDir(), t.TempDir(), []string{"../etc"})
	if err == nil {
		t.Fatal("expected escape error")
	}
}

func TestCopyDirsSkipsMissing(t *testing.T) {
	if err := CopyDirs(context.Background(), t.TempDir(), t.TempDir(), []string{"node_modules"}); err != nil {
		t.Fatal(err)
	}
}
