package remap

import (
	"context"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

//go:embed testdata/httpserver.c.src
var httpServerSrc string

//go:embed testdata/publicconnect.c.src
var publicConnectSrc string

// CompileHTTPServer builds an unsigned libc HTTP server for remap tests.
// System interpreters on macOS are SIP-protected and drop DYLD_INSERT_LIBRARIES.
func CompileHTTPServer(ctx context.Context, dest string) error {
	return compileC(ctx, dest, "httpserver.c", httpServerSrc)
}

// CompilePublicConnect builds an unsigned libc client that dials 1.1.1.1:443.
func CompilePublicConnect(ctx context.Context, dest string) error {
	return compileC(ctx, dest, "publicconnect.c", publicConnectSrc)
}

func compileC(ctx context.Context, dest, name, src string) error {
	cc, err := compiler()
	if err != nil {
		return err
	}
	dir, err := os.MkdirTemp("", "lane-remap-fixture-*")
	if err != nil {
		return err
	}
	defer func() { _ = os.RemoveAll(dir) }()
	path := filepath.Join(dir, name)
	if err := os.WriteFile(path, []byte(src), 0o644); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(dest), 0o755); err != nil {
		return err
	}
	cmd := exec.CommandContext(ctx, cc, "-O2", "-o", dest, path)
	out, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("compile %s: %w\n%s", name, err, out)
	}
	unsign(dest)
	return nil
}
