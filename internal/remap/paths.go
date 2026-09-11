package remap

import (
	"path/filepath"

	"github.com/Mrjwj34/lane/internal/config"
	"github.com/Mrjwj34/lane/internal/home"
)

func LibPath() string  { return filepath.Join(home.BinDir(), libName()) }
func WrapPath() string { return filepath.Join(home.BinDir(), "lane-remap-wrap") }

func TablePath(worktree string) string {
	return filepath.Join(config.LaneDir(worktree), "remap.txt")
}

func PIDPath(worktree string) string {
	return filepath.Join(config.LaneDir(worktree), "remap.pid")
}

func LogPath(worktree string) string {
	return filepath.Join(config.LaneDir(worktree), "remap.log")
}
