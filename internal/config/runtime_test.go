package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestRuntimeContracts(t *testing.T) {
	cases := []struct {
		text string
		ok   bool
	}{
		{"version: 1\nports: [web]\n", true},
		{"version: 1\nports: [web]\nlisten: {web: 30000}\nruntime: {backend: container, image: berth-test:local}\n", true},
		{"version: 1\nports: [web]\nruntime: {backend: container, image: berth-test:local}\n", false},
		{"version: 1\nports: [my-api, my_api]\n", false},
		{"version: 1\nports: [pc]\n", false},
		{"version: 1\nisolate: net\n", false},
		{"version: 1\nenv_file: ../outside\n", false},
		{"version: 1\ncopy_dirs: [.berth]\n", false},
		{"version: 1\nenv: {BERTH_DATA_DIR: /shared}\n", false},
	}
	for _, tc := range cases {
		path := filepath.Join(t.TempDir(), Filename)
		if err := os.WriteFile(path, []byte(tc.text), 0o644); err != nil {
			t.Fatal(err)
		}
		_, err := Load(path)
		if (err == nil) != tc.ok {
			t.Fatalf("config %q: %v (want valid=%v)", tc.text, err, tc.ok)
		}
	}
}
