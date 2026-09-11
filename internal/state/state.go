package state

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/Mrjwj34/berth/internal/config"
	"github.com/Mrjwj34/berth/internal/home"
	"github.com/Mrjwj34/berth/internal/ports"
	"github.com/gofrs/flock"
)

const version = 2

const Owned = "berth-created"
const Adopted = "adopted"

// File is the on-disk machine-level registry.
type File struct {
	Version    int                  `json:"version"`
	Workspaces map[string]Workspace `json:"workspaces"`
}

// Workspace is identified by its absolute directory path.
type Workspace struct {
	Ownership     string         `json:"ownership,omitempty"`
	GitDir        string         `json:"git_dir,omitempty"`
	Phase         string         `json:"phase,omitempty"`
	SetupComplete bool           `json:"setup_complete,omitempty"`
	ResetPending  bool           `json:"reset_pending,omitempty"`
	RemovalHead   string         `json:"removal_head,omitempty"`
	LastError     string         `json:"last_error,omitempty"`
	Runtime       config.Runtime `json:"runtime,omitempty"`
	Listen        map[string]int `json:"listen,omitempty"`
	ID            string         `json:"id"`
	Slug          string         `json:"slug"`
	Path          string         `json:"path"`
	Repo          string         `json:"repo"`
	Branch        string         `json:"branch"`
	Base          string         `json:"base,omitempty"`
	Ports         map[string]int `json:"ports,omitempty"`
	CreatedAt     time.Time      `json:"created_at"`
	LastUsedAt    time.Time      `json:"last_used_at"`
}

// Store is a locked, atomically-updated view of ~/.berth/state.json.
type Store struct {
	path string
	mu   sync.Mutex
	lock *flock.Flock
}

func Open(_ context.Context) (*Store, error) {
	dir := home.Dir()
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return nil, fmt.Errorf("create berth home %s: %w", dir, err)
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
	ok, err := s.lock.TryRLockContext(ctx, 20*time.Millisecond)
	if err == nil && !ok {
		err = ctx.Err()
	}
	if err != nil {
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
	ok, err := s.lock.TryLockContext(ctx, 20*time.Millisecond)
	if err == nil && !ok {
		err = ctx.Err()
	}
	if err != nil {
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
		return nil, fmt.Errorf("parse %s: %w. Run berth doctor --fix", s.path, err)
	}
	if f.Version > version || f.Version < 0 {
		return nil, fmt.Errorf("unsupported state version %d", f.Version)
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
	if ev, err := filepath.EvalSymlinks(abs); err == nil {
		abs = ev
	}
	abs = filepath.Clean(abs)
	if runtime.GOOS == "windows" {
		abs = strings.ToLower(abs)
	}
	return abs, nil
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

// Reserve assigns ports and records the workspace in one cross-process
// transaction. Kernel bind ownership remains with the actual runtime.
func (s *Store) Reserve(ctx context.Context, ws Workspace, names []string) (Workspace, error) {
	err := s.Update(ctx, func(f *File) error {
		if _, ok := f.Workspaces[ws.Path]; ok {
			return fmt.Errorf("workspace path already registered: %s", ws.Path)
		}
		if _, ok := f.BySlug(ws.Repo, ws.Slug); ok {
			return fmt.Errorf("workspace slug already registered: %s", ws.Slug)
		}
		allocated, err := ports.Allocate(ctx, f.UsedPorts(), names)
		if err != nil {
			return err
		}
		ws.Ports = allocated
		f.Workspaces[ws.Path] = ws
		return nil
	})
	return ws, err
}

// LockWorkspace survives checkout removal. It serializes lifecycle operations
// without holding the machine-wide registry lock while running user commands.
func (s *Store) LockWorkspace(ctx context.Context, path string) (func(), error) {
	key, err := Key(path)
	if err != nil {
		return nil, err
	}
	sum := sha256.Sum256([]byte(key))
	dir := filepath.Join(filepath.Dir(s.path), "locks")
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return nil, err
	}
	lock := flock.New(filepath.Join(dir, hex.EncodeToString(sum[:])+".lock"))
	ok, err := lock.TryLockContext(ctx, 20*time.Millisecond)
	if err != nil {
		return nil, err
	}
	if !ok {
		return nil, ctx.Err()
	}
	return func() { _ = lock.Unlock() }, nil
}
