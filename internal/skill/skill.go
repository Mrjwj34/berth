package skill

import (
	"embed"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
)

//go:embed SKILL.md hooks/*
var assets embed.FS

func Content() ([]byte, error) {
	return assets.ReadFile("SKILL.md")
}

func Install(repoRoot string) error {
	targets := []string{
		filepath.Join(repoRoot, ".agents", "skills", "lane", "SKILL.md"),
		filepath.Join(repoRoot, ".claude", "skills", "lane", "SKILL.md"),
		filepath.Join(repoRoot, ".cursor", "skills", "lane", "SKILL.md"),
	}
	data, err := Content()
	if err != nil {
		return fmt.Errorf("embed SKILL.md: %w", err)
	}
	for _, dest := range targets {
		if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(dest, data, 0o644); err != nil {
			return fmt.Errorf("write %s: %w", dest, err)
		}
	}
	return nil
}

func HookAsset(name string) ([]byte, error) {
	return assets.ReadFile(filepath.ToSlash(filepath.Join("hooks", name)))
}

func WriteHooks(repoRoot string) error {
	destDir := filepath.Join(repoRoot, ".lane", "hooks")
	if err := os.MkdirAll(destDir, 0o755); err != nil {
		return err
	}
	return fs.WalkDir(assets, "hooks", func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		data, err := assets.ReadFile(path)
		if err != nil {
			return err
		}
		dest := filepath.Join(destDir, filepath.Base(path))
		mode := os.FileMode(0o644)
		if filepath.Ext(dest) == ".sh" {
			mode = 0o755
		}
		return os.WriteFile(dest, data, mode)
	})
}
