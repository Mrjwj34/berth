package gc

import (
	"context"
	"path/filepath"
	"testing"
	"time"

	"github.com/Mrjwj34/berth/internal/state"
)

func TestReclaimVanished(t *testing.T) {
	t.Setenv("BERTH_HOME", t.TempDir())
	ctx := context.Background()
	st, err := state.Open(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ghost := filepath.Join(t.TempDir(), "gone")
	if err := st.Update(ctx, func(f *state.File) error {
		f.Workspaces[ghost] = state.Workspace{Slug: "gone", Path: ghost, Repo: t.TempDir(), LastUsedAt: time.Now()}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	rep, err := Run(ctx, st, Options{DryRun: false})
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Actions) != 1 || rep.Actions[0].Kind != "reclaim" {
		t.Fatalf("actions = %+v", rep.Actions)
	}
}
