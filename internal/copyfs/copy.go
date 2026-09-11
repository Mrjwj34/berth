package copyfs

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
)

// CopyDirs copies each relative directory from srcRoot to dstRoot.
// macOS uses clonefile when available; otherwise files are hardlinked,
// falling back to a regular copy.
func CopyDirs(ctx context.Context, srcRoot, dstRoot string, dirs []string) error {
	for _, dir := range dirs {
		if err := ctx.Err(); err != nil {
			return err
		}
		rel := filepath.Clean(dir)
		if rel == ".." || stringsHasDotDot(rel) {
			return fmt.Errorf("copy_dirs: path %q escapes the repository", dir)
		}
		src := filepath.Join(srcRoot, rel)
		dst := filepath.Join(dstRoot, rel)
		st, err := os.Stat(src)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return fmt.Errorf("copy_dirs %s: %w", dir, err)
		}
		if !st.IsDir() {
			return fmt.Errorf("copy_dirs %s is not a directory", dir)
		}
		if err := copyTree(src, dst); err != nil {
			return fmt.Errorf("copy_dirs %s: %w", dir, err)
		}
	}
	return nil
}

func stringsHasDotDot(rel string) bool {
	for _, p := range filepath.SplitList(rel) {
		_ = p
	}
	for _, p := range splitPath(rel) {
		if p == ".." {
			return true
		}
	}
	return false
}

func splitPath(p string) []string {
	var out []string
	for p != "." && p != string(filepath.Separator) && p != "" {
		out = append([]string{filepath.Base(p)}, out...)
		next := filepath.Dir(p)
		if next == p {
			break
		}
		p = next
	}
	return out
}

func copyTree(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			_ = os.Remove(target)
			return os.Symlink(link, target)
		}
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		return copyFile(path, target, info.Mode().Perm())
	})
}

func copyFile(src, dst string, perm os.FileMode) error {
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	_ = os.Remove(dst)
	if runtime.GOOS == "darwin" {
		if err := clonefile(src, dst); err == nil {
			return nil
		}
	}
	if err := os.Link(src, dst); err == nil {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, perm)
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}
