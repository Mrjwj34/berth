package state

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"
)

func TestReserveChild(t *testing.T) {
	name := os.Getenv("BERTH_RESERVE_HELPER")
	if name == "" {
		t.Skip("subprocess helper")
	}
	st, err := Open(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	_, err = st.Reserve(context.Background(), Workspace{ID: name, Slug: name, Path: filepath.Join(os.Getenv("BERTH_HOME"), name), Repo: "repo"}, []string{"web", "db"})
	if err != nil {
		t.Fatal(err)
	}
}
func TestReserveAcrossProcesses(t *testing.T) {
	t.Setenv("BERTH_HOME", t.TempDir())
	exe, err := os.Executable()
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	var cmds []*exec.Cmd
	for i := 0; i < 8; i++ {
		cmd := exec.CommandContext(ctx, exe, "-test.run=^TestReserveChild$")
		cmd.Env = append(os.Environ(), fmt.Sprintf("BERTH_RESERVE_HELPER=ws-%d", i))
		if err := cmd.Start(); err != nil {
			t.Fatal(err)
		}
		cmds = append(cmds, cmd)
	}
	for _, cmd := range cmds {
		if err := cmd.Wait(); err != nil {
			t.Fatal(err)
		}
	}
	st, err := Open(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	f, err := st.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Workspaces) != 8 || len(f.UsedPorts()) != 16 {
		t.Fatalf("duplicate port reservations: %+v", f.Workspaces)
	}
}
func TestLegacyOwnershipIsNotUpgraded(t *testing.T) {
	t.Setenv("BERTH_HOME", t.TempDir())
	st, err := Open(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	if err := os.WriteFile(st.path, []byte(`{"version":1,"workspaces":{"/old":{"id":"old","path":"/old"}}}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := st.Update(context.Background(), func(f *File) error {
		if f.Workspaces["/old"].Ownership != "" {
			t.Fatal("legacy deletion authority granted")
		}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	f, err := st.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if f.Version != version || f.Workspaces["/old"].Ownership != "" {
		t.Fatal("unsafe migration")
	}
}
