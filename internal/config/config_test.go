package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadAndExpand(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, Filename)
	content := `
version: 1
base: main
ports: [web, api]
env:
  PORT: ${LANE_PORT_API}
  URL: http://127.0.0.1:$LANE_PORT_WEB
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Base != "main" || len(cfg.Ports) != 2 {
		t.Fatalf("unexpected cfg: %+v", cfg)
	}
	id := IdentityVars("/ws", "s", "/repo", "lane/s", map[string]int{"web": 20000, "api": 20001})
	merged := MergeEnv(id, cfg.Env)
	if merged["PORT"] != "20001" || merged["URL"] != "http://127.0.0.1:20000" {
		t.Fatalf("expand failed: %+v", merged)
	}
	if PortEnvName("my-web") != "LANE_PORT_MY_WEB" {
		t.Fatalf("port env name: %s", PortEnvName("my-web"))
	}
}

func TestWriteEnvFileManagedBlock(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".env.local")
	if err := os.WriteFile(path, []byte("KEEP=1\n# BEGIN LANE\nOLD=x\n# END LANE\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteEnvFile(path, map[string]string{"LANE_PORT_WEB": "1", "FOO": "bar"}); err != nil {
		t.Fatal(err)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	s := string(got)
	if !strings.Contains(s, "KEEP=1") || strings.Contains(s, "OLD=x") {
		t.Fatalf("managed block mishandled:\n%s", s)
	}
	if !strings.Contains(s, EnvBegin) || !strings.Contains(s, "LANE_PORT_WEB=1") {
		t.Fatalf("missing managed vars:\n%s", s)
	}
}

func TestFindMissingUsesDefaults(t *testing.T) {
	dir := t.TempDir()
	cfg, root, err := Find(dir)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Base != "main" || root != dir {
		t.Fatalf("cfg=%+v root=%s", cfg, root)
	}
}

func TestLoadListenPorts(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, Filename)
	content := `
version: 1
ports:
  api: 8080
  web:
    listen: 5173
  extra:
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.Ports.NeedsIsolate() {
		t.Fatal("expected isolate")
	}
	got := cfg.Ports.ListenMap()
	if got["api"] != 8080 || got["web"] != 5173 {
		t.Fatalf("listen map: %+v", got)
	}
	names := cfg.Ports.Names()
	if len(names) != 3 {
		t.Fatalf("names: %v", names)
	}
	if ListenEnvName("api") != "LANE_LISTEN_API" {
		t.Fatalf("listen env: %s", ListenEnvName("api"))
	}
}

func TestLoadIsolateNet(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, Filename)
	if err := os.WriteFile(path, []byte("version: 1\nisolate: net\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if !cfg.IsolateNet() || cfg.Isolate != "net" {
		t.Fatalf("cfg: %+v", cfg)
	}
}

func TestLoadListenPortListObjects(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, Filename)
	content := `
version: 1
ports:
  - name: api
    listen: 8080
  - web
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(cfg.Ports) != 2 || cfg.Ports[0].Listen != 8080 || cfg.Ports[1].Name != "web" {
		t.Fatalf("ports: %+v", cfg.Ports)
	}
}

func TestDuplicatePortsRejected(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, Filename)
	if err := os.WriteFile(path, []byte("version: 1\nports: [web, WEB]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(path); err == nil {
		t.Fatal("expected duplicate port error")
	}
}
