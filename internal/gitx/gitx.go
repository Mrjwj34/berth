package gitx

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
)

func Run(ctx context.Context, dir string, args ...string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Dir = dir
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		msg := strings.TrimSpace(stderr.String())
		if msg == "" {
			msg = err.Error()
		}
		return "", fmt.Errorf("git %s: %s", strings.Join(args, " "), msg)
	}
	return strings.TrimSpace(stdout.String()), nil
}

func TopLevel(ctx context.Context, dir string) (string, error) {
	return Run(ctx, dir, "rev-parse", "--show-toplevel")
}

func CommonDir(ctx context.Context, dir string) (string, error) {
	return Run(ctx, dir, "rev-parse", "--path-format=absolute", "--git-common-dir")
}

// MainRepo returns the primary checkout directory (the repo that owns .git).
func MainRepo(ctx context.Context, dir string) (string, error) {
	common, err := CommonDir(ctx, dir)
	if err != nil {
		return "", err
	}
	if filepath.Base(common) == ".git" {
		return filepath.Dir(common), nil
	}
	return TopLevel(ctx, dir)
}

func CurrentBranch(ctx context.Context, dir string) (string, error) {
	return Run(ctx, dir, "rev-parse", "--abbrev-ref", "HEAD")
}

func Has(ctx context.Context) error {
	if _, err := exec.LookPath("git"); err != nil {
		return fmt.Errorf("git is not on PATH. Install Git and retry")
	}
	_, err := Run(ctx, "", "version")
	return err
}

func IsInsideWorkTree(ctx context.Context, dir string) bool {
	out, err := Run(ctx, dir, "rev-parse", "--is-inside-work-tree")
	return err == nil && out == "true"
}

func DefaultBranch(ctx context.Context, dir string) string {
	for _, name := range []string{"main", "master"} {
		if _, err := Run(ctx, dir, "rev-parse", "--verify", name); err == nil {
			return name
		}
		if _, err := Run(ctx, dir, "rev-parse", "--verify", "origin/"+name); err == nil {
			return name
		}
	}
	if b, err := CurrentBranch(ctx, dir); err == nil && b != "HEAD" {
		return b
	}
	return "main"
}
