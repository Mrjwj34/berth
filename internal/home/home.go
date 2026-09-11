package home

import (
	"os"
	"path/filepath"
)

// Dir returns the machine-level berth home directory.
// BERTH_HOME overrides the default ~/.berth so tests never touch the user profile.
func Dir() string {
	if v := os.Getenv("BERTH_HOME"); v != "" {
		return v
	}
	user, err := os.UserHomeDir()
	if err != nil || user == "" {
		return ".berth"
	}
	return filepath.Join(user, ".berth")
}

func StatePath() string { return filepath.Join(Dir(), "state.json") }
func LockPath() string  { return filepath.Join(Dir(), "state.lock") }
func BinDir() string    { return filepath.Join(Dir(), "bin") }
