package home

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDirRespectsLANEHOME(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("LANE_HOME", dir)
	if got := Dir(); got != dir {
		t.Fatalf("Dir() = %q, want %q", got, dir)
	}
	if StatePath() != filepath.Join(dir, "state.json") {
		t.Fatalf("StatePath() = %q", StatePath())
	}
	if LockPath() != filepath.Join(dir, "state.lock") {
		t.Fatalf("LockPath() = %q", LockPath())
	}
	if BinDir() != filepath.Join(dir, "bin") {
		t.Fatalf("BinDir() = %q", BinDir())
	}
}

func TestDirDefaultUsesHome(t *testing.T) {
	user := t.TempDir()
	t.Setenv("HOME", user)
	t.Setenv("USERPROFILE", user)
	_ = os.Unsetenv("LANE_HOME")
	got := Dir()
	if got != filepath.Join(user, ".lane") {
		t.Fatalf("Dir() = %q, want %s/.lane", got, user)
	}
}
