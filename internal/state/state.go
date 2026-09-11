package state

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Mrjwj34/lane/internal/home"
	"github.com/gofrs/flock"
)

const version = 1

// File is the on-disk machine-level registry.
type File struct {
	Version    int                  `json:"version"`
	Workspaces map[string]Workspace `json:"workspaces"`
}

// Workspace is identified by its absolute directory path.
type Workspace struct {
	ID         string         `json:"id"`
	Slug       string         `json:"slug"`
	Path       string         `json:"path"`
	Repo       string         `json:"repo"`
	Branch     string         `json:"branch"`
	Base       string         `json:"base,omitempty"`
	Ports      map[string]int `json:"ports,omitempty"`
	CreatedAt  time.Time      `json:"created_at"`
	LastUsedAt time.Time      `json:"last_used_at"`
}

// Store is a locked, atomically-updated view of ~/.lane/state.json.
type Store struct {
	path string
	mu   sync.Mutex
	lock *flock.Flock
}

func Open(_ context.Context) (*Store, error) {
	dir := home.Dir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create lane home %s: %w", dir, err)
	}
	lock := flock.New(home.LockPath())
	return &Store{path: home.StatePath(), lock: lock}, nil
}

func (s *Store) Close() error {
	if s == nil || s.lock == nil {
		return nil
	}
	return s.lock.Unlock()
}

func (s *Store) Read(ctx context.Context) (*File, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.lock.RLock(); err != nil {
		return nil, fmt.Errorf("lock state for read: %w", err)
	}
	defer func() { _ = s.lock.Unlock() }()
	return s.readUnlocked()
}

func (s *Store) Update(ctx context.Context, fn func(*File) error) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := s.lock.Lock(); err != nil {
		return fmt.Errorf("lock state for write: %w", err)
	}
	defer func() { _ = s.lock.Unlock() }()

	f, err := s.readUnlocked()
	if err != nil {
		return err
	}
	if err := fn(f); err != nil {
		return err
	}
	return s.writeUnlocked(f)
}

func (s *Store) readUnlocked() (*File, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return &File{Version: version, Workspaces: map[string]Workspace{}}, nil
		}
		return nil, fmt.Errorf("read %s: %w", s.path, err)
	}
	var f File
	if err := json.Unmarshal(data, &f); err != nil {
		return nil, fmt.Errorf("parse %s: %w. Run lane doctor --fix", s.path, err)
	}
	if f.Workspaces == nil {
		f.Workspaces = map[string]Workspace{}
	}
	if f.Version == 0 {
		f.Version = version
	}
	return &f, nil
}

func (s *Store) writeUnlocked(f *File) error {
	if f.Workspaces == nil {
		f.Workspaces = map[string]Workspace{}
	}
	f.Version = version
	data, err := json.MarshalIndent(f, "", "  ")
	if err != nil {
		return fmt.Errorf("encode state: %w", err)
	}
	data = append(data, '\n')
	dir := filepath.Dir(s.path)
	tmp, err := os.CreateTemp(dir, "state-*.tmp")
	if err != nil {
		return fmt.Errorf("create temp state: %w", err)
	}
	tmpName := tmp.Name()
	_, writeErr := tmp.Write(data)
	syncErr := tmp.Sync()
	closeErr := tmp.Close()
	if writeErr != nil || syncErr != nil || closeErr != nil {
		_ = os.Remove(tmpName)
		if writeErr != nil {
			return fmt.Errorf("write temp state: %w", writeErr)
		}
		if syncErr != nil {
			return fmt.Errorf("sync temp state: %w", syncErr)
		}
		return fmt.Errorf("close temp state: %w", closeErr)
	}
	if err := os.Rename(tmpName, s.path); err != nil {
		_ = os.Remove(tmpName)
		return fmt.Errorf("atomic replace %s: %w", s.path, err)
	}
	return nil
}

func Key(path string) (string, error) {
	abs, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	return filepath.Clean(abs), nil
}

func (f *File) Lookup(path string) (Workspace, bool) {
	key, err := Key(path)
	if err != nil {
		return Workspace{}, false
	}
	ws, ok := f.Workspaces[key]
	return ws, ok
}

func (f *File) BySlug(repo, slug string) (Workspace, bool) {
	repoKey, err := Key(repo)
	if err != nil {
		return Workspace{}, false
	}
	for _, ws := range f.Workspaces {
		if ws.Slug == slug {
			wsRepo, err := Key(ws.Repo)
			if err == nil && wsRepo == repoKey {
				return ws, true
			}
		}
	}
	return Workspace{}, false
}

func (f *File) UsedPorts() map[int]string {
	used := map[int]string{}
	for _, ws := range f.Workspaces {
		for name, port := range ws.Ports {
			used[port] = ws.Path + ":" + name
		}
	}
	return used
}

func Touch(ws *Workspace) {
	ws.LastUsedAt = time.Now().UTC()
}
