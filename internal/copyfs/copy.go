package copyfs

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Mrjwj34/berth/internal/config"
)

// CopyDirs makes independent writable copies. CoW is an optimization; hard
// links are never a fallback because modifications would cross workspaces.
func CopyDirs(ctx context.Context, srcRoot, dstRoot string, dirs []string) error {
	for _, dir := range dirs {
		if err := CopyPath(ctx, srcRoot, dstRoot, dir); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("copy_dirs %s: %w", dir, err)
		}
	}
	return nil
}
func within(root, path string) bool {
	rel, err := filepath.Rel(root, path)
	return err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator))
}

// CopyPath is shared by copy_dirs and .worktreeinclude. Root-level symlinks and
// escaping links are rejected instead of silently sharing mutable state.
func CopyPath(ctx context.Context, srcRoot, dstRoot, rel string) error {
	if err := config.RelativePath(rel); err != nil {
		return err
	}
	src, err := filepath.EvalSymlinks(srcRoot)
	if err != nil {
		return err
	}
	dst, err := filepath.EvalSymlinks(dstRoot)
	if err != nil {
		return err
	}
	src, err = filepath.Abs(src)
	if err != nil {
		return err
	}
	dst, err = filepath.Abs(dst)
	if err != nil {
		return err
	}
	if within(src, dst) || within(dst, src) {
		return fmt.Errorf("copy roots must be distinct and non-nested")
	}
	from, to := filepath.Join(src, rel), filepath.Join(dst, rel)
	// Validate source ancestors too; a parent symlink can otherwise bypass Walk.
	if err := safeParents(src, filepath.Dir(from)); err != nil {
		return err
	}
	return filepath.Walk(from, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if err := ctx.Err(); err != nil {
			return err
		}
		part, err := filepath.Rel(from, path)
		if err != nil {
			return err
		}
		target := filepath.Join(to, part)
		if err := safeParents(dst, filepath.Dir(target)); err != nil {
			return err
		}
		if info.IsDir() {
			if old, err := os.Lstat(target); err == nil && (!old.IsDir() || old.Mode()&os.ModeSymlink != 0) {
				return fmt.Errorf("unsafe directory %s", target)
			}
			return os.MkdirAll(target, info.Mode().Perm())
		}
		if info.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(path)
			if err != nil {
				return err
			}
			resolved := link
			if !filepath.IsAbs(link) {
				resolved = filepath.Join(filepath.Dir(path), link)
			}
			final, err := filepath.EvalSymlinks(resolved)
			if err != nil {
				return fmt.Errorf("resolve symlink %s: %w", path, err)
			}
			if !within(src, final) || !within(src, resolved) {
				return fmt.Errorf("external symlink %s must be installed separately", path)
			}
			if filepath.IsAbs(link) {
				part, err := filepath.Rel(src, link)
				if err != nil {
					return err
				}
				link = filepath.Join(dst, part)
			}
			if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
				return err
			}
			if err := os.Remove(target); err != nil && !os.IsNotExist(err) {
				return err
			}
			return os.Symlink(link, target)
		}
		if !info.Mode().IsRegular() {
			return fmt.Errorf("unsupported file type: %s", path)
		}
		return copyFile(path, target, info.Mode().Perm())
	})
}
func safeParents(root, path string) error {
	for within(root, path) {
		st, err := os.Lstat(path)
		if err != nil && !os.IsNotExist(err) {
			return err
		}
		if err == nil && st.Mode()&os.ModeSymlink != 0 {
			return fmt.Errorf("copy parent is a symlink: %s", path)
		}
		if path == root {
			return nil
		}
		next := filepath.Dir(path)
		if next == path {
			break
		}
		path = next
	}
	return nil
}
func copyFile(src, dst string, perm os.FileMode) error {
	original, err := os.Stat(src)
	if err != nil {
		return err
	}
	if existing, err := os.Stat(dst); err == nil && os.SameFile(original, existing) {
		return fmt.Errorf("refusing to copy a file onto itself: %s", src)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	if err := os.Remove(dst); err != nil && !os.IsNotExist(err) {
		return err
	}
	if runtime.GOOS == "darwin" {
		if err := clonefile(src, dst); err == nil {
			return nil
		}
	}
	if err := reflink(src, dst, perm); err == nil {
		return nil
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_EXCL|os.O_WRONLY, perm)
	if err != nil {
		return err
	}
	_, err = io.Copy(out, in)
	closeErr := out.Close()
	if err != nil {
		_ = os.Remove(dst)
		return err
	}
	return closeErr
}
