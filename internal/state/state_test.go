package state

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestReadMissingFile(t *testing.T) {
	t.Setenv("BERTH_HOME", t.TempDir())
	st, err := Open(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	f, err := st.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if f.Version != version || len(f.Workspaces) != 0 {
		t.Fatalf("unexpected empty state: %+v", f)
	}
}

func TestUpdateRoundTrip(t *testing.T) {
	t.Setenv("BERTH_HOME", t.TempDir())
	st, err := Open(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	ctx := context.Background()
	if err := st.Update(ctx, func(f *File) error {
		f.Workspaces["/tmp/ws"] = Workspace{ID: "1", Slug: "a", Path: "/tmp/ws", Repo: "/tmp/repo", Branch: "berth/a", CreatedAt: time.Now().UTC()}
		return nil
	}); err != nil {
		t.Fatal(err)
	}
	f, err := st.Read(ctx)
	if err != nil {
		t.Fatal(err)
	}
	ws, ok := f.Workspaces["/tmp/ws"]
	if !ok || ws.Slug != "a" {
		t.Fatalf("missing workspace: %+v", f.Workspaces)
	}
}

func TestConcurrentUpdates(t *testing.T) {
	t.Setenv("BERTH_HOME", t.TempDir())
	const n = 20
	var wg sync.WaitGroup
	wg.Add(n)
	errCh := make(chan error, n)
	for i := 0; i < n; i++ {
		i := i
		go func() {
			defer wg.Done()
			st, err := Open(context.Background())
			if err != nil {
				errCh <- err
				return
			}
			defer st.Close()
			err = st.Update(context.Background(), func(f *File) error {
				key := fmt.Sprintf("/ws/%d", i)
				f.Workspaces[key] = Workspace{ID: key, Slug: fmt.Sprintf("s%d", i), Path: key, Repo: "/repo"}
				return nil
			})
			if err != nil {
				errCh <- err
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		t.Fatal(err)
	}
	st, err := Open(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	defer st.Close()
	f, err := st.Read(context.Background())
	if err != nil {
		t.Fatal(err)
	}
	if len(f.Workspaces) != n {
		t.Fatalf("got %d workspaces, want %d", len(f.Workspaces), n)
	}
}
