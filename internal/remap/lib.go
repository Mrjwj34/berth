package remap

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
)

func sourceHash() string {
	sum := sha256.Sum256([]byte(preloadSrc + "\n" + wrapSrc + "\n" + launchSrc + "\n" + runtime.GOOS + "\n" + runtime.GOARCH))
	return hex.EncodeToString(sum[:12])
}

func hashPath(lib string) string { return lib + ".hash" }

func libFresh(lib string) bool {
	st, err := os.Stat(lib)
	if err != nil || st.IsDir() || st.Size() == 0 {
		return false
	}
	got, err := os.ReadFile(hashPath(lib))
	if err != nil {
		return false
	}
	return strings.TrimSpace(string(got)) == sourceHash()
}

func compiler() (string, error) {
	for _, name := range []string{"cc", "gcc", "clang"} {
		if p, err := exec.LookPath(name); err == nil {
			return p, nil
		}
	}
	return "", fmt.Errorf("no C compiler (cc) on PATH. Install a compiler so lane can build the bind remap library, or run a single workspace")
}

// EnsureLib compiles the intercept library into $LANE_HOME/bin when missing or stale.
func EnsureLib(ctx context.Context) (string, error) {
	if runtime.GOOS == "windows" {
		return "", fmt.Errorf("%w", errUnsupported)
	}
	lib := LibPath()
	if libFresh(lib) && wrapFresh() && launchFresh() {
		return lib, nil
	}
	if err := ctx.Err(); err != nil {
		return "", err
	}
	cc, err := compiler()
	if err != nil {
		if st, stErr := os.Stat(lib); stErr == nil && !st.IsDir() && st.Size() > 0 {
			return lib, nil
		}
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(lib), 0o755); err != nil {
		return "", err
	}
	dir, err := os.MkdirTemp("", "lane-remap-*")
	if err != nil {
		return "", err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	src := filepath.Join(dir, "preload.c")
	if err := os.WriteFile(src, []byte(preloadSrc), 0o644); err != nil {
		return "", err
	}
	tmp := filepath.Join(dir, libName())
	var args []string
	if runtime.GOOS == "darwin" {
		args = []string{"-dynamiclib", "-fPIC", "-O2", "-install_name", lib, "-o", tmp, src}
	} else {
		args = []string{"-shared", "-fPIC", "-O2", "-o", tmp, src, "-ldl"}
	}
	cmd := exec.CommandContext(ctx, cc, args...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("compile remap library: %w\n%s", err, out)
	}
	data, err := os.ReadFile(tmp)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(lib, data, 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(hashPath(lib), []byte(sourceHash()+"\n"), 0o644); err != nil {
		return "", err
	}
	if err := compileWrap(ctx, cc, dir); err != nil {
		return "", err
	}
	if err := compileLaunch(ctx, cc, dir); err != nil {
		return "", err
	}
	// Keep the compiler's ad-hoc signature on the dylib. Stripping it
	// makes AMFI refuse DYLD_INSERT_LIBRARIES on modern macOS.
	return lib, nil
}

func wrapFresh() bool {
	if runtime.GOOS != "linux" {
		return true
	}
	st, err := os.Stat(WrapPath())
	return err == nil && !st.IsDir() && st.Size() > 0
}

func launchFresh() bool {
	if runtime.GOOS == "windows" {
		return true
	}
	st, err := os.Stat(LaunchPath())
	return err == nil && !st.IsDir() && st.Size() > 0
}

func compileWrap(ctx context.Context, cc, dir string) error {
	if runtime.GOOS != "linux" {
		return nil
	}
	src := filepath.Join(dir, "wrap.c")
	if err := os.WriteFile(src, []byte(wrapSrc), 0o644); err != nil {
		return err
	}
	tmp := filepath.Join(dir, "lane-remap-wrap")
	cmd := exec.CommandContext(ctx, cc, "-O2", "-o", tmp, src)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("compile remap wrapper: %w\n%s", err, out)
	}
	data, err := os.ReadFile(tmp)
	if err != nil {
		return err
	}
	if err := os.WriteFile(WrapPath(), data, 0o755); err != nil {
		return err
	}
	unsign(WrapPath())
	return nil
}

func compileLaunch(ctx context.Context, cc, dir string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	src := filepath.Join(dir, "launch.c")
	if err := os.WriteFile(src, []byte(launchSrc), 0o644); err != nil {
		return err
	}
	tmp := filepath.Join(dir, "lane-remap-launch")
	cmd := exec.CommandContext(ctx, cc, "-O2", "-o", tmp, src)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("compile remap launcher: %w\n%s", err, out)
	}
	data, err := os.ReadFile(tmp)
	if err != nil {
		return err
	}
	if err := os.WriteFile(LaunchPath(), data, 0o755); err != nil {
		return err
	}
	unsign(LaunchPath())
	return nil
}

func unsign(path string) {
	if runtime.GOOS != "darwin" {
		return
	}
	cs, err := exec.LookPath("codesign")
	if err != nil {
		return
	}
	_ = exec.Command(cs, "--remove-signature", path).Run()
}

func Available() bool {
	if runtime.GOOS != "linux" && runtime.GOOS != "darwin" {
		return false
	}
	if libFresh(LibPath()) {
		return true
	}
	_, err := compiler()
	return err == nil
}
