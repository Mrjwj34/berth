package worktree

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/Mrjwj34/lane/internal/gitx"
)

const IncludeFile = ".worktreeinclude"
const BranchPrefix = "lane/"

type Info struct {
	Path   string
	Branch string
	Bare   bool
}

func DefaultPath(repo, slug string) string {
	parent := filepath.Dir(repo)
	base := filepath.Base(repo)
	return filepath.Join(parent, base+".lanes", slug)
}

func ResolvePath(repo, slug, worktreeRoot string) string {
	if worktreeRoot == "" {
		return DefaultPath(repo, slug)
	}
	if filepath.IsAbs(worktreeRoot) {
		return filepath.Join(worktreeRoot, slug)
	}
	return filepath.Join(repo, worktreeRoot, slug)
}

func BranchName(slug string) string {
	slug = strings.TrimPrefix(slug, BranchPrefix)
	return BranchPrefix + slug
}

func ValidateSlug(slug string) error {
	if slug == "" {
		return fmt.Errorf("slug is empty. Use a lowercase name like feat-login")
	}
	if strings.Contains(slug, "/") || strings.Contains(slug, "\\") {
		return fmt.Errorf("slug %q must not contain slashes", slug)
	}
	for _, r := range slug {
		ok := (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_'
		if !ok {
			return fmt.Errorf("slug %q must be lowercase ASCII, digits, '-' or '_'", slug)
		}
	}
	return nil
}

func Add(ctx context.Context, repo, path, branch, startPoint string) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return fmt.Errorf("create worktree parent: %w", err)
	}
	if _, err := os.Stat(path); err == nil {
		return fmt.Errorf("worktree path %s already exists. Use lane attach or choose another slug", path)
	}
	if _, err := gitx.Run(ctx, repo, "rev-parse", "--verify", startPoint); err != nil {
		return fmt.Errorf("base branch %q not found. Set base: in lane.yaml or pass --base", startPoint)
	}
	if _, err := gitx.Run(ctx, repo, "show-ref", "--verify", "--quiet", "refs/heads/"+branch); err == nil {
		_, err = gitx.Run(ctx, repo, "worktree", "add", path, branch)
		if err != nil {
			return fmt.Errorf("git worktree add: %w", err)
		}
		return nil
	}
	_, err := gitx.Run(ctx, repo, "worktree", "add", "-b", branch, path, startPoint)
	if err != nil {
		return fmt.Errorf("git worktree add: %w", err)
	}
	return nil
}

func Remove(ctx context.Context, repo, path string, force bool) error {
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, path)
	if _, err := gitx.Run(ctx, repo, args...); err != nil {
		if force {
			if rmErr := os.RemoveAll(path); rmErr != nil {
				return fmt.Errorf("remove worktree %s: %w (also: %v)", path, err, rmErr)
			}
			_, _ = gitx.Run(ctx, repo, "worktree", "prune")
			return nil
		}
		return fmt.Errorf("worktree is dirty. Commit changes or use --force: %w", err)
	}
	_, _ = gitx.Run(ctx, repo, "worktree", "prune")
	return nil
}

func DeleteBranch(ctx context.Context, repo, branch string) error {
	if branch == "" {
		return nil
	}
	_, err := gitx.Run(ctx, repo, "branch", "-D", branch)
	return err
}

func List(ctx context.Context, repo string) ([]Info, error) {
	out, err := gitx.Run(ctx, repo, "worktree", "list", "--porcelain")
	if err != nil {
		return nil, err
	}
	var infos []Info
	var cur Info
	flush := func() {
		if cur.Path != "" {
			infos = append(infos, cur)
		}
		cur = Info{}
	}
	sc := bufio.NewScanner(strings.NewReader(out))
	for sc.Scan() {
		line := sc.Text()
		switch {
		case strings.HasPrefix(line, "worktree "):
			flush()
			cur.Path = strings.TrimPrefix(line, "worktree ")
		case strings.HasPrefix(line, "branch "):
			ref := strings.TrimPrefix(line, "branch ")
			cur.Branch = strings.TrimPrefix(ref, "refs/heads/")
		case line == "bare":
			cur.Bare = true
		case line == "":
			flush()
		}
	}
	flush()
	return infos, sc.Err()
}

func Dirty(ctx context.Context, path string) (bool, string, error) {
	out, err := gitx.Run(ctx, path, "status", "--porcelain")
	if err != nil {
		return false, "", err
	}
	return out != "", out, nil
}

func Unpublished(ctx context.Context, path string) (bool, string, error) {
	branch, err := gitx.CurrentBranch(ctx, path)
	if err != nil {
		return false, "", err
	}
	if branch == "HEAD" {
		return true, "detached HEAD", nil
	}
	upstream, err := gitx.Run(ctx, path, "rev-parse", "--abbrev-ref", "--symbolic-full-name", "@{upstream}")
	if err != nil {
		return true, "no upstream (push the branch or use --force)", nil
	}
	counts, err := gitx.Run(ctx, path, "rev-list", "--left-right", "--count", upstream+"...HEAD")
	if err != nil {
		return false, "", err
	}
	var behind, ahead int
	if _, scanErr := fmt.Sscanf(counts, "%d\t%d", &behind, &ahead); scanErr != nil {
		fmt.Sscanf(counts, "%d %d", &behind, &ahead)
	}
	if ahead > 0 {
		return true, fmt.Sprintf("%d commit(s) not pushed to %s", ahead, upstream), nil
	}
	return false, "", nil
}

func MergedInto(ctx context.Context, repo, branch, base string) (bool, error) {
	if _, err := gitx.Run(ctx, repo, "rev-parse", "--verify", branch); err != nil {
		return false, nil
	}
	if _, err := gitx.Run(ctx, repo, "merge-base", "--is-ancestor", branch, base); err != nil {
		if _, err2 := gitx.Run(ctx, repo, "merge-base", "--is-ancestor", branch, "origin/"+base); err2 != nil {
			return false, nil
		}
	}
	return true, nil
}

func ApplyInclude(ctx context.Context, srcRepo, dest string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	includePath := filepath.Join(srcRepo, IncludeFile)
	data, err := os.ReadFile(includePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("read %s: %w", includePath, err)
	}
	sc := bufio.NewScanner(strings.NewReader(string(data)))
	for sc.Scan() {
		line := strings.TrimSpace(sc.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		rel := filepath.Clean(line)
		if rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
			return fmt.Errorf("%s: path %q escapes the repository", IncludeFile, line)
		}
		src := filepath.Join(srcRepo, rel)
		dst := filepath.Join(dest, rel)
		if err := copyPath(src, dst); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("copy %s: %w", rel, err)
		}
	}
	return sc.Err()
}

func copyPath(src, dst string) error {
	st, err := os.Lstat(src)
	if err != nil {
		return err
	}
	if st.IsDir() {
		return copyDirPlain(src, dst)
	}
	if err := os.MkdirAll(filepath.Dir(dst), 0o755); err != nil {
		return err
	}
	in, err := os.Open(src)
	if err != nil {
		return err
	}
	defer in.Close()
	out, err := os.OpenFile(dst, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, st.Mode().Perm())
	if err != nil {
		return err
	}
	defer out.Close()
	_, err = io.Copy(out, in)
	return err
}

func copyDirPlain(src, dst string) error {
	return filepath.Walk(src, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, info.Mode().Perm())
		}
		return copyPath(path, target)
	})
}
