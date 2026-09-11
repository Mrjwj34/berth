package worktree

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/Mrjwj34/lane/internal/copyfs"
	"github.com/Mrjwj34/lane/internal/gitx"
)

const IncludeFile = ".worktreeinclude"
const BranchPrefix = "lane/"

type Info struct {
	Path   string
	Branch string
	Bare   bool
}

func SamePath(a, b string) bool {
	ca, err1 := canon(a)
	cb, err2 := canon(b)
	if err1 != nil || err2 != nil {
		return filepath.Clean(a) == filepath.Clean(b)
	}
	if runtime.GOOS == "windows" {
		return strings.EqualFold(ca, cb)
	}
	return ca == cb
}

func canon(p string) (string, error) {
	abs, err := filepath.Abs(p)
	if err != nil {
		return "", err
	}
	if ev, err := filepath.EvalSymlinks(abs); err == nil {
		abs = ev
	}
	return filepath.Clean(abs), nil
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

// ValidateTarget protects the primary checkout and verifies Git ownership even
// when --force is requested. Force never authorizes an arbitrary RemoveAll.
func ValidateTarget(ctx context.Context, repo, path string) error {
	if SamePath(repo, path) {
		return fmt.Errorf("refusing to operate on the primary worktree")
	}
	p, err := canon(path)
	if err != nil {
		return err
	}
	r, err := canon(repo)
	if err != nil {
		return err
	}
	rel, err := filepath.Rel(p, r)
	if err == nil && rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return fmt.Errorf("target is an ancestor of the primary repository")
	}
	st, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !st.IsDir() || st.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("worktree must be a real directory")
	}
	top, err := gitx.TopLevel(ctx, path)
	if err != nil {
		return err
	}
	if !SamePath(top, path) {
		return fmt.Errorf("target is not a worktree root")
	}
	common, err := gitx.CommonDir(ctx, path)
	if err != nil {
		return err
	}
	owner, err := gitx.CommonDir(ctx, repo)
	if err != nil {
		return err
	}
	if !SamePath(common, owner) {
		return fmt.Errorf("worktree belongs to a different repository")
	}
	infos, err := List(ctx, repo)
	if err != nil {
		return err
	}
	for _, info := range infos {
		if !info.Bare && SamePath(info.Path, path) {
			return nil
		}
	}
	return fmt.Errorf("target is not a registered linked worktree")
}
func Remove(ctx context.Context, repo, path string, force bool) error {
	if err := ValidateTarget(ctx, repo, path); err != nil {
		return err
	}
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, "--", path)
	if _, err := gitx.Run(ctx, repo, args...); err != nil {
		return fmt.Errorf("git worktree remove (no filesystem fallback): %w", err)
	}
	return nil
}

func DeleteBranch(ctx context.Context, repo, branch string) error {
	if branch == "" {
		return nil
	}
	_, err := gitx.Run(ctx, repo, "branch", "-D", "--", branch)
	return err
}

func List(ctx context.Context, repo string) ([]Info, error) {
	out, err := gitx.Run(ctx, repo, "worktree", "list", "--porcelain", "-z")
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
	for _, line := range strings.Split(out, "\x00") {
		switch {
		case strings.HasPrefix(line, "worktree "):
			flush()
			cur.Path = strings.TrimPrefix(line, "worktree ")
		case strings.HasPrefix(line, "branch "):
			cur.Branch = strings.TrimPrefix(strings.TrimPrefix(line, "branch "), "refs/heads/")
		case line == "bare":
			cur.Bare = true
		case line == "":
			flush()
		}
	}
	flush()
	return infos, nil
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
	if n, scanErr := fmt.Sscanf(counts, "%d %d", &behind, &ahead); scanErr != nil || n != 2 {
		return false, "", fmt.Errorf("invalid Git commit counts %q", counts)
	}
	if ahead > 0 {
		return true, fmt.Sprintf("%d commit(s) not pushed to %s", ahead, upstream), nil
	}
	return false, "", nil
}

func MergedInto(ctx context.Context, repo, branch, base string) (bool, error) {
	if base == "" {
		return false, fmt.Errorf("merge base is not configured")
	}
	if _, err := gitx.Run(ctx, repo, "rev-parse", "--verify", "--end-of-options", branch+"^{commit}"); err != nil {
		return false, err
	}
	for _, ref := range []string{base, "origin/" + base} {
		if _, err := gitx.Run(ctx, repo, "rev-parse", "--verify", "--end-of-options", ref+"^{commit}"); err != nil {
			if ctx.Err() != nil {
				return false, ctx.Err()
			}
			continue
		}
		if _, err := gitx.Run(ctx, repo, "merge-base", "--is-ancestor", branch, ref); err == nil {
			return true, nil
		} else {
			var exit *exec.ExitError
			if !errors.As(err, &exit) || exit.ExitCode() != 1 {
				return false, err
			}
		}
	}
	return false, nil
}

// Preserved verifies actual HEAD, not a stale registered branch. A clean tree
// alone is insufficient: every commit must be merged or present upstream.
func Preserved(ctx context.Context, repo, path, base string) error {
	dirty, detail, err := Dirty(ctx, path)
	if err != nil {
		return err
	}
	if dirty {
		return fmt.Errorf("worktree is dirty; commit changes first:\n%s", detail)
	}
	head, err := gitx.Run(ctx, path, "rev-parse", "HEAD")
	if err != nil {
		return err
	}
	merged, err := MergedInto(ctx, repo, head, base)
	if err != nil {
		return err
	}
	if merged {
		return nil
	}
	unpublished, why, err := Unpublished(ctx, path)
	if err != nil {
		return err
	}
	if unpublished {
		return fmt.Errorf("work is not preserved: %s; push or merge it before removal", why)
	}
	return nil
}
func ApplyInclude(ctx context.Context, srcRepo, dest string) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	data, err := os.ReadFile(filepath.Join(srcRepo, IncludeFile))
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	sc := bufio.NewScanner(strings.NewReader(string(data)))
	for sc.Scan() {
		rel := strings.TrimSpace(sc.Text())
		if rel == "" || strings.HasPrefix(rel, "#") {
			continue
		}
		if err := copyfs.CopyPath(ctx, srcRepo, dest, rel); err != nil && !os.IsNotExist(err) {
			return fmt.Errorf("include %s: %w", rel, err)
		}
	}
	return sc.Err()
}
