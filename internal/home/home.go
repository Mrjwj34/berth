package home

import (
	"os"
	"path/filepath"
)

// Dir returns the machine-level lane home directory.
// LANE_HOME overrides the default ~/.lane so tests never touch the user profile.
func Dir() string {
	if v := os.Getenv("LANE_HOME"); v != "" {
		return v
	}
	user, err := os.UserHomeDir()
	if err != nil || user == "" {
		return ".lane"
	}
	return filepath.Join(user, ".lane")
}

func StatePath() string { return filepath.Join(Dir(), "state.json") }
func LockPath() string  { return filepath.Join(Dir(), "state.lock") }
func BinDir() string    { return filepath.Join(Dir(), "bin") }
