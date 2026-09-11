package process

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/Mrjwj34/lane/internal/home"
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
	return filepath.Join(home.BinDir(), name)
}

func LookPath() (string, error) {
	pinned := BinPath()
	if st, err := os.Stat(pinned); err == nil && !st.IsDir() {
		return pinned, nil
	}
	if p, err := exec.LookPath("process-compose"); err == nil {
		return p, nil
	}
	return "", fmt.Errorf("process-compose not found. Run lane doctor --fix to download %s", PinnedVersion)
}

func Ensure(ctx context.Context) (string, error) {
	if p, err := LookPath(); err == nil {
		return p, nil
	}
	return Download(ctx)
}

func Download(ctx context.Context) (string, error) {
	asset, err := releaseAsset()
	if err != nil {
		return "", err
	}
	url := fmt.Sprintf("%s/%s/%s", releaseBase, PinnedVersion, asset)
	if err := os.MkdirAll(home.BinDir(), 0o755); err != nil {
		return "", err
	}
	tmp := filepath.Join(home.BinDir(), asset+".download")
	if err := fetch(ctx, url, tmp); err != nil {
		return "", fmt.Errorf("download process-compose %s: %w. Check network or install process-compose %s onto PATH", PinnedVersion, err, PinnedVersion)
	}
	defer os.Remove(tmp)
	dest := BinPath()
	if strings.HasSuffix(asset, ".zip") {
		if err := extractZip(tmp, dest); err != nil {
			return "", err
		}
	} else {
		if err := extractTarGz(tmp, dest); err != nil {
			return "", err
		}
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(dest, 0o755); err != nil {
			return "", err
		}
	}
	return dest, nil
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
	_, err = io.Copy(f, resp.Body)
	return err
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
	cmd := exec.CommandContext(ctx, bin, "version")
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
