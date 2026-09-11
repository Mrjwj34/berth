package copyfs

import (
	"context"
	"os"
	"path/filepath"
	"testing"
)

func TestIndependentWritableCopies(t *testing.T) {
	src, dst := t.TempDir(), t.TempDir()
	if err := os.Mkdir(filepath.Join(src, "deps"), 0o755); err != nil {
		t.Fatal(err)
	}
	a, b := filepath.Join(src, "deps", "file"), filepath.Join(dst, "deps", "file")
	if err := os.WriteFile(a, []byte("original"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CopyDirs(context.Background(), src, dst, []string{"deps"}); err != nil {
		t.Fatal(err)
	}
	sa, _ := os.Stat(a)
	sb, _ := os.Stat(b)
	if os.SameFile(sa, sb) {
		t.Fatal("copies share inode")
	}
	if err := os.WriteFile(b, []byte("modified"), 0o644); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(a)
	if err != nil || string(data) != "original" {
		t.Fatalf("source contaminated: %q %v", data, err)
	}
}
func TestSameRootPreservesSource(t *testing.T) {
	src := t.TempDir()
	if err := os.Mkdir(filepath.Join(src, "deps"), 0o755); err != nil {
		t.Fatal(err)
	}
	a := filepath.Join(src, "deps", "file")
	if err := os.WriteFile(a, []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CopyDirs(context.Background(), src, src, []string{"deps"}); err == nil {
		t.Fatal("same-root copy accepted")
	}
	if data, err := os.ReadFile(a); err != nil || string(data) != "keep" {
		t.Fatal("source deleted")
	}
	if err := copyFile(a, a, 0o644); err == nil {
		t.Fatal("same-file copy accepted")
	}
}
