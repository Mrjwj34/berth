package process

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Mrjwj34/berth/internal/home"
	"github.com/gofrs/flock"
)

const (
	PinnedVersion = "v1.122.0"
	releaseBase   = "https://github.com/F1bonacc1/process-compose/releases/download"
)

func BinPath() string {
	name := "process-compose"
	if runtime.GOOS == "windows" {
		name += ".exe"
	}
	return filepath.Join(home.BinDir(), "process-compose-"+PinnedVersion, name)
}

func LookPath() (string, error) {
	pinned := BinPath()
	if st, err := os.Stat(pinned); err == nil && !st.IsDir() {
		return pinned, nil
	}
	if p, err := exec.LookPath("process-compose"); err == nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		version, err := Version(ctx, p)
		if err == nil && strings.TrimPrefix(version, "v") == strings.TrimPrefix(PinnedVersion, "v") {
			return p, nil
		}
	}
	return "", fmt.Errorf("process-compose not found. Run berth doctor --fix to download %s", PinnedVersion)
}

func Ensure(ctx context.Context) (string, error) {
	if p, err := LookPath(); err == nil {
		return p, nil
	}
	return Download(ctx)
}

// Checksums manifest digest is pinned alongside the release version, rather
// than trusting an independently downloaded mutable checksum manifest.
const checksumManifestSHA256 = "07e5377c2224380a1cf9fe2741e587f58f4051e532389ab7c7edec0ff2445684"

func Download(ctx context.Context) (string, error) {
	asset, err := releaseAsset()
	if err != nil {
		return "", err
	}
	dir := filepath.Dir(BinPath())
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return "", err
	}
	lock := flock.New(filepath.Join(dir, "install.lock"))
	ok, err := lock.TryLockContext(ctx, 25*time.Millisecond)
	if err != nil {
		return "", err
	}
	if !ok {
		return "", ctx.Err()
	}
	defer func() { _ = lock.Unlock() }()
	if st, err := os.Stat(BinPath()); err == nil && st.Mode().IsRegular() && st.Size() > 0 {
		return BinPath(), nil
	}
	tmp, err := os.MkdirTemp(dir, "install-*")
	if err != nil {
		return "", err
	}
	defer func() { _ = os.RemoveAll(tmp) }()
	base := fmt.Sprintf("%s/%s/", releaseBase, PinnedVersion)
	manifest := filepath.Join(tmp, "checksums.txt")
	if err := fetch(ctx, base+"process-compose_checksums.txt", manifest); err != nil {
		return "", err
	}
	if err := verifySHA256(manifest, checksumManifestSHA256); err != nil {
		return "", err
	}
	data, err := os.ReadFile(manifest)
	if err != nil {
		return "", err
	}
	expected := ""
	for _, line := range strings.Split(string(data), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 && strings.TrimPrefix(fields[1], "*") == asset {
			expected = fields[0]
			break
		}
	}
	if len(expected) != 64 {
		return "", fmt.Errorf("release has no checksum for %s", asset)
	}
	archive := filepath.Join(tmp, asset)
	if err := fetch(ctx, base+asset, archive); err != nil {
		return "", err
	}
	if err := verifySHA256(archive, expected); err != nil {
		return "", err
	}
	target := filepath.Join(tmp, filepath.Base(BinPath()))
	if strings.HasSuffix(asset, ".zip") {
		err = extractZip(archive, target)
	} else {
		err = extractTarGz(archive, target)
	}
	if err != nil {
		return "", err
	}
	if err := os.Chmod(target, 0o755); err != nil {
		return "", err
	}
	version, err := Version(ctx, target)
	if err != nil {
		return "", err
	}
	if strings.TrimPrefix(version, "v") != strings.TrimPrefix(PinnedVersion, "v") {
		return "", fmt.Errorf("unexpected process-compose version %s", version)
	}
	if err := os.Rename(target, BinPath()); err != nil {
		return "", err
	}
	return BinPath(), nil
}
func verifySHA256(path, want string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	defer f.Close()
	hash := sha256.New()
	if _, err := io.Copy(hash, f); err != nil {
		return err
	}
	if got := hex.EncodeToString(hash.Sum(nil)); got != want {
		return fmt.Errorf("checksum mismatch for %s: got %s", filepath.Base(path), got)
	}
	return nil
}

func releaseAsset() (string, error) {
	goos := runtime.GOOS
	arch := runtime.GOARCH
	switch arch {
	case "amd64", "arm64", "386", "arm":
	default:
		return "", fmt.Errorf("unsupported architecture %s for process-compose", arch)
	}
	if goos == "windows" {
		return fmt.Sprintf("process-compose_windows_%s.zip", arch), nil
	}
	return fmt.Sprintf("process-compose_%s_%s.tar.gz", goos, arch), nil
}

func fetch(ctx context.Context, url, dest string) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	client := &http.Client{Timeout: 2 * time.Minute}
	resp, err := client.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("HTTP %s", resp.Status)
	}
	f, err := os.Create(dest)
	if err != nil {
		return err
	}
	defer f.Close()
	n, err := io.Copy(f, io.LimitReader(resp.Body, 128<<20))
	if err != nil {
		return err
	}
	if n >= 128<<20 {
		return fmt.Errorf("download exceeds size limit")
	}
	return f.Sync()
}

func extractTarGz(src, dest string) error {
	f, err := os.Open(src)
	if err != nil {
		return err
	}
	defer f.Close()
	gz, err := gzip.NewReader(f)
	if err != nil {
		return err
	}
	defer gz.Close()
	tr := tar.NewReader(gz)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}
		base := filepath.Base(hdr.Name)
		if hdr.Typeflag == tar.TypeReg && (base == "process-compose" || base == "process-compose.exe") {
			out, err := os.Create(dest)
			if err != nil {
				return err
			}
			_, err = io.Copy(out, tr)
			_ = out.Close()
			return err
		}
	}
	return fmt.Errorf("process-compose binary not found in archive")
}

func extractZip(src, dest string) error {
	zr, err := zip.OpenReader(src)
	if err != nil {
		return err
	}
	defer zr.Close()
	for _, f := range zr.File {
		base := filepath.Base(f.Name)
		if base == "process-compose.exe" || base == "process-compose" {
			rc, err := f.Open()
			if err != nil {
				return err
			}
			out, err := os.Create(dest)
			if err != nil {
				rc.Close()
				return err
			}
			_, err = io.Copy(out, rc)
			rc.Close()
			_ = out.Close()
			return err
		}
	}
	return fmt.Errorf("process-compose binary not found in zip")
}

func Version(ctx context.Context, bin string) (string, error) {
	child, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()
	cmd := exec.CommandContext(child, bin, "version")
	cmd.Env = append(os.Environ(), "PC_DISABLE_TUI=1")
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("process-compose version: %w", err)
	}
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "Version:") {
			return strings.TrimSpace(strings.TrimPrefix(line, "Version:")), nil
		}
	}
	return strings.TrimSpace(string(out)), nil
}
