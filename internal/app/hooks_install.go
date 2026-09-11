package app

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/Mrjwj34/berth/internal/skill"
)

// The hook files berth merges into. They belong to the harness and may already
// hold the user's own entries, so berth never replaces a key wholesale: it
// decodes the file, appends its own entry only when no berth entry is present,
// and preserves every other key and value.
const (
	cursorHookFile   = ".cursor/worktrees.json"
	windsurfHookFile = ".windsurf/hooks.json"
	claudeHookFile   = ".claude/settings.json"

	cursorEvent   = "setup-worktree"
	windsurfEvent = "post_setup_worktree"
	claudeEvent   = "WorktreeCreate"
)

// adoptCommand is what Cursor and Windsurf run inside a worktree they created.
const adoptCommand = "berth adopt --setup"

// claudeAdoptCommand always exits 0. Claude Code aborts worktree creation on
// any non-zero exit from a WorktreeCreate hook, so a failed adoption must not
// fail the hook: berth writes its diagnosis to stderr and `berth gc` reclaims
// the checkout that was left unregistered. `|| exit 0` is understood by sh,
// cmd.exe and PowerShell 7, which is all berth can rely on without knowing
// which shell the harness uses.
const claudeAdoptCommand = adoptCommand + " || exit 0"

// windsurfEntry is appended to the post_setup_worktree array. Windsurf runs
// `command` through bash on macOS and Linux and `powershell` on Windows, so both
// are set to the same argument-preserving command.
type windsurfEntry struct {
	Command    string `json:"command"`
	Powershell string `json:"powershell"`
	ShowOutput bool   `json:"show_output"`
}

type claudeHandler struct {
	Type    string `json:"type"`
	Command string `json:"command"`
}

type claudeGroup struct {
	Hooks []claudeHandler `json:"hooks"`
}

func ensureGitignore(repo string) error {
	path := filepath.Join(repo, ".gitignore")
	data, err := os.ReadFile(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	s := string(data)
	if strings.Contains(s, ".berth/") {
		return nil
	}
	var b strings.Builder
	b.WriteString(s)
	if s != "" && !strings.HasSuffix(s, "\n") {
		b.WriteByte('\n')
	}
	b.WriteString("\n# berth workspaces\n.berth/\n")
	return os.WriteFile(path, []byte(b.String()), 0o644)
}

// writeCursorHook merges the Cursor adapter into .cursor/worktrees.json. Cursor
// runs those commands inside every worktree it creates, so those worktrees get
// registered with berth instead of being recreated by it.
//
// The file is shared with the user: Cursor's documentation puts unrelated setup
// steps such as `npm ci` in the same arrays, so berth appends its command only
// when the array has no berth entry and leaves everything else untouched.
func writeCursorHook(repo string) (bool, error) {
	data, err := skill.Adapter("cursor.worktrees.json")
	if err != nil {
		return false, err
	}
	var want map[string]json.RawMessage
	if err := json.Unmarshal(data, &want); err != nil {
		return false, fmt.Errorf("decode the embedded Cursor adapter: %w", err)
	}
	return mergeJSONObject(repo, cursorHookFile, func(obj map[string]json.RawMessage) (bool, error) {
		changed := false
		for _, key := range sortedKeys(want) {
			have, err := decodeArray(obj[key], cursorHookFile, key)
			if err != nil {
				return false, err
			}
			if hasEntry(have, skill.HookMarker) {
				continue
			}
			add, err := decodeArray(want[key], cursorHookFile, key)
			if err != nil {
				return false, err
			}
			if len(add) == 0 {
				continue
			}
			merged, err := encodeArray(append(append([]json.RawMessage{}, have...), add...))
			if err != nil {
				return false, err
			}
			obj[key] = merged
			changed = true
		}
		return changed, nil
	})
}

// writeWindsurfHook appends berth to the post_setup_worktree array in
// .windsurf/hooks.json, creating the file when it is absent. Windsurf merges
// hook files across system, user and workspace scope, so an entry added here
// does not displace anyone else's.
func writeWindsurfHook(repo string) (bool, error) {
	return mergeJSONObject(repo, windsurfHookFile, func(obj map[string]json.RawMessage) (bool, error) {
		hooks := map[string]json.RawMessage{}
		if raw := obj["hooks"]; len(raw) > 0 {
			if err := json.Unmarshal(raw, &hooks); err != nil {
				return false, fmt.Errorf("%s: key \"hooks\" is not a JSON object, so berth will not replace it. Fix the file and re-run: %w", windsurfHookFile, err)
			}
		}
		have, err := decodeArray(hooks[windsurfEvent], windsurfHookFile, windsurfEvent)
		if err != nil {
			return false, err
		}
		if hasEntry(have, skill.HookMarker) {
			return false, nil
		}
		entry, err := json.Marshal(windsurfEntry{Command: adoptCommand, Powershell: adoptCommand, ShowOutput: true})
		if err != nil {
			return false, err
		}
		merged, err := encodeArray(append(append([]json.RawMessage{}, have...), entry))
		if err != nil {
			return false, err
		}
		hooks[windsurfEvent] = merged
		encoded, err := json.Marshal(hooks)
		if err != nil {
			return false, err
		}
		obj["hooks"] = encoded
		return true, nil
	})
}

// writeClaudeHook adds a WorktreeCreate hook to .claude/settings.json as a
// sibling of the existing event keys, which is the merge discipline Claude Code
// documents. It is installed only when claude is named explicitly: the hook
// replaces nothing, but a non-zero exit from it aborts worktree creation, so
// berth's adapter there always exits 0.
func writeClaudeHook(repo string) (bool, error) {
	return mergeJSONObject(repo, claudeHookFile, func(obj map[string]json.RawMessage) (bool, error) {
		hooks := map[string]json.RawMessage{}
		if raw := obj["hooks"]; len(raw) > 0 {
			if err := json.Unmarshal(raw, &hooks); err != nil {
				return false, fmt.Errorf("%s: key \"hooks\" is not a JSON object, so berth will not replace it. Fix the file and re-run: %w", claudeHookFile, err)
			}
		}
		have, err := decodeArray(hooks[claudeEvent], claudeHookFile, claudeEvent)
		if err != nil {
			return false, err
		}
		if hasEntry(have, skill.HookMarker) {
			return false, nil
		}
		group, err := json.Marshal(claudeGroup{Hooks: []claudeHandler{{Type: "command", Command: claudeAdoptCommand}}})
		if err != nil {
			return false, err
		}
		merged, err := encodeArray(append(append([]json.RawMessage{}, have...), group))
		if err != nil {
			return false, err
		}
		hooks[claudeEvent] = merged
		encoded, err := json.Marshal(hooks)
		if err != nil {
			return false, err
		}
		obj["hooks"] = encoded
		return true, nil
	})
}

// mergeJSONObject reads the JSON object at repo/rel, lets mutate edit it, and
// writes the result back. Nothing is written when mutate reports no change, so
// a repeated install leaves the user's file byte-for-byte identical. A file
// berth cannot parse is an error, never a target to overwrite.
func mergeJSONObject(repo, rel string, mutate func(map[string]json.RawMessage) (bool, error)) (bool, error) {
	path := filepath.Join(repo, filepath.FromSlash(rel))
	obj := map[string]json.RawMessage{}
	switch existing, err := os.ReadFile(path); {
	case err == nil:
		if err := json.Unmarshal(existing, &obj); err != nil {
			return false, fmt.Errorf("cannot merge berth's hook into %s: %w. Fix the JSON and re-run; berth never overwrites a file it cannot parse", rel, err)
		}
	case os.IsNotExist(err):
	default:
		return false, fmt.Errorf("read %s: %w", rel, err)
	}
	changed, err := mutate(obj)
	if err != nil {
		return false, err
	}
	if !changed {
		return false, nil
	}
	merged, err := json.MarshalIndent(obj, "", "  ")
	if err != nil {
		return false, fmt.Errorf("encode %s: %w", rel, err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return false, fmt.Errorf("create %s: %w", filepath.Dir(path), err)
	}
	if err := writeAtomic(path, append(merged, '\n')); err != nil {
		return false, fmt.Errorf("write %s: %w", rel, err)
	}
	return true, nil
}

// decodeArray reads a JSON array of elements that are kept verbatim. A missing
// key is an empty array; a key that exists but is not an array is an error, so
// berth reports a file it cannot merge instead of replacing it.
func decodeArray(raw json.RawMessage, rel, key string) ([]json.RawMessage, error) {
	if len(raw) == 0 {
		return nil, nil
	}
	var arr []json.RawMessage
	if err := json.Unmarshal(raw, &arr); err != nil {
		return nil, fmt.Errorf("%s: key %q is not a JSON array, so berth will not replace it. Fix the file and re-run", rel, key)
	}
	return arr, nil
}

func encodeArray(arr []json.RawMessage) (json.RawMessage, error) {
	return json.Marshal(arr)
}

// hasEntry reports whether an array already contains a berth entry. Elements
// are matched as strings when they are strings (Cursor's command lists) and by
// substring otherwise (Windsurf and Claude entries are objects), so a berth
// entry added by hand still suppresses a duplicate.
func hasEntry(elems []json.RawMessage, marker string) bool {
	for _, elem := range elems {
		var s string
		if json.Unmarshal(elem, &s) == nil {
			if strings.Contains(s, marker) {
				return true
			}
			continue
		}
		if bytes.Contains(elem, []byte(marker)) {
			return true
		}
	}
	return false
}

func sortedKeys(m map[string]json.RawMessage) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// writeAtomic replaces path through a temporary file and a rename. No harness
// documents a locking or partial-write contract for third-party hook
// installation, so a reader must never see a half-written file.
func writeAtomic(path string, data []byte) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp*")
	if err != nil {
		return err
	}
	name := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		_ = tmp.Close()
		_ = os.Remove(name)
		return err
	}
	if err := tmp.Close(); err != nil {
		_ = os.Remove(name)
		return err
	}
	if err := os.Chmod(name, 0o644); err != nil {
		_ = os.Remove(name)
		return err
	}
	if err := os.Rename(name, path); err != nil {
		_ = os.Remove(name)
		return err
	}
	return nil
}
