package app

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Mrjwj34/berth/internal/skill"
)

func writeFile(t *testing.T, path, body string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// object decodes a hook file, so assertions are about the merge semantics
// rather than about formatting.
func object(t *testing.T, path string) map[string]any {
	t.Helper()
	var out map[string]any
	if err := json.Unmarshal([]byte(readFile(t, path)), &out); err != nil {
		t.Fatalf("%s is not a JSON object: %v", path, err)
	}
	return out
}

func array(t *testing.T, value any, context string) []any {
	t.Helper()
	list, ok := value.([]any)
	if !ok {
		t.Fatalf("%s is not an array: %#v", context, value)
	}
	return list
}

func TestCursorHookAppendsWithoutLosingUserEntries(t *testing.T) {
	repo := t.TempDir()
	path := filepath.Join(repo, ".cursor", "worktrees.json")
	writeFile(t, path, `{
  "version": 2,
  "setup-worktree-unix": [
    "npm ci",
    "cp $ROOT_WORKTREE_PATH/.env .env"
  ],
  "berth-note": "hand written key berth knows nothing about"
}
`)
	changed, err := writeCursorHook(repo)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("first merge reported no change")
	}
	first := readFile(t, path)
	root := object(t, path)
	if root["berth-note"] != "hand written key berth knows nothing about" {
		t.Fatalf("an unknown top-level key was dropped: %v", root)
	}
	if root["version"] != float64(2) {
		t.Fatalf("an existing value was rewritten: %v", root["version"])
	}
	unix := array(t, root["setup-worktree-unix"], "setup-worktree-unix")
	if len(unix) != 3 {
		t.Fatalf("setup-worktree-unix = %v, want the two user commands plus berth", unix)
	}
	if unix[0] != "npm ci" || unix[1] != "cp $ROOT_WORKTREE_PATH/.env .env" {
		t.Fatalf("existing commands changed: %v", unix)
	}
	if unix[2] != "berth adopt --setup" {
		t.Fatalf("berth entry = %v", unix[2])
	}
	for _, key := range []string{"setup-worktree-windows", "setup-worktree"} {
		entries := array(t, root[key], key)
		if len(entries) != 1 || entries[0] != "berth adopt --setup" {
			t.Fatalf("%s = %v", key, entries)
		}
	}

	changed, err = writeCursorHook(repo)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("second merge rewrote the file")
	}
	if again := readFile(t, path); again != first {
		t.Fatalf("second merge changed the file:\n%s\nthen\n%s", first, again)
	}
}

func TestCursorHookCreatesTheFileWhenAbsent(t *testing.T) {
	repo := t.TempDir()
	changed, err := writeCursorHook(repo)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("expected a change")
	}
	root := object(t, filepath.Join(repo, ".cursor", "worktrees.json"))
	if len(root) != 3 {
		t.Fatalf("keys = %v, want the three setup-worktree keys", root)
	}
}

// TestCursorHookRecognisesAHandWrittenBerthEntry covers the append-if-absent
// rule: an entry the user wrote themselves is not duplicated, even when they
// wrapped it in a longer command.
func TestCursorHookRecognisesAHandWrittenBerthEntry(t *testing.T) {
	repo := t.TempDir()
	path := filepath.Join(repo, ".cursor", "worktrees.json")
	writeFile(t, path, `{
  "setup-worktree": ["berth adopt --setup"],
  "setup-worktree-unix": ["berth adopt --setup && npm test"],
  "setup-worktree-windows": ["berth adopt --setup"]
}
`)
	before := readFile(t, path)
	changed, err := writeCursorHook(repo)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("berth duplicated its own entry")
	}
	if after := readFile(t, path); after != before {
		t.Fatal("the file was rewritten even though nothing was missing")
	}
	unix := array(t, object(t, path)["setup-worktree-unix"], "setup-worktree-unix")
	if len(unix) != 1 {
		t.Fatalf("setup-worktree-unix = %v", unix)
	}
}

func TestCursorHookRefusesToClobberWhatItCannotMerge(t *testing.T) {
	cases := []struct {
		name string
		body string
		want string
	}{
		{name: "invalid JSON", body: "{ not json", want: "never overwrites"},
		{name: "key is not an array", body: `{"setup-worktree": {"command": "npm ci"}}`, want: "not a JSON array"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			repo := t.TempDir()
			path := filepath.Join(repo, ".cursor", "worktrees.json")
			writeFile(t, path, tc.body)
			_, err := writeCursorHook(repo)
			if err == nil || !strings.Contains(err.Error(), tc.want) {
				t.Fatalf("error = %v, want %q", err, tc.want)
			}
			if got := readFile(t, path); got != tc.body {
				t.Fatalf("the file was modified: %q", got)
			}
		})
	}
}

func TestWindsurfHookAppendsWithoutDuplicating(t *testing.T) {
	repo := t.TempDir()
	path := filepath.Join(repo, ".windsurf", "hooks.json")
	writeFile(t, path, `{
  "hooks": {
    "pre_run_command": [{ "command": "echo hi" }],
    "post_setup_worktree": [{ "command": "bash $ROOT_WORKSPACE_PATH/hooks/setup.sh", "show_output": true }]
  }
}
`)
	changed, err := writeWindsurfHook(repo)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("first merge reported no change")
	}
	first := readFile(t, path)
	root := object(t, path)
	hooks, ok := root["hooks"].(map[string]any)
	if !ok {
		t.Fatalf("hooks = %#v", root["hooks"])
	}
	if len(array(t, hooks["pre_run_command"], "pre_run_command")) != 1 {
		t.Fatal("another event's hook was dropped")
	}
	entries := array(t, hooks["post_setup_worktree"], "post_setup_worktree")
	if len(entries) != 2 {
		t.Fatalf("post_setup_worktree = %v, want the user entry plus berth", entries)
	}
	user, ok := entries[0].(map[string]any)
	if !ok || user["command"] != "bash $ROOT_WORKSPACE_PATH/hooks/setup.sh" {
		t.Fatalf("the user entry changed: %#v", entries[0])
	}
	berth, ok := entries[1].(map[string]any)
	if !ok {
		t.Fatalf("berth entry = %#v", entries[1])
	}
	if berth["command"] != "berth adopt --setup" || berth["powershell"] != "berth adopt --setup" || berth["show_output"] != true {
		t.Fatalf("berth entry = %#v", berth)
	}

	changed, err = writeWindsurfHook(repo)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("second merge rewrote the file")
	}
	if again := readFile(t, path); again != first {
		t.Fatal("second merge changed the file")
	}
}

func TestWindsurfHookCreatesTheFileWhenAbsent(t *testing.T) {
	repo := t.TempDir()
	if _, err := writeWindsurfHook(repo); err != nil {
		t.Fatal(err)
	}
	root := object(t, filepath.Join(repo, ".windsurf", "hooks.json"))
	hooks, ok := root["hooks"].(map[string]any)
	if !ok {
		t.Fatalf("hooks = %#v", root["hooks"])
	}
	entries := array(t, hooks["post_setup_worktree"], "post_setup_worktree")
	if len(entries) != 1 {
		t.Fatalf("post_setup_worktree = %v", entries)
	}
}

func TestClaudeHookAddsASiblingEventKey(t *testing.T) {
	repo := t.TempDir()
	path := filepath.Join(repo, ".claude", "settings.json")
	writeFile(t, path, `{
  "permissions": { "allow": ["Bash(git status)"] },
  "hooks": {
    "PreToolUse": [{ "matcher": "Bash", "hooks": [{ "type": "command", "command": "./check.sh" }] }]
  }
}
`)
	changed, err := writeClaudeHook(repo)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("first merge reported no change")
	}
	first := readFile(t, path)
	root := object(t, path)
	if _, ok := root["permissions"].(map[string]any); !ok {
		t.Fatalf("an unrelated top-level key was dropped: %v", root)
	}
	hooks, ok := root["hooks"].(map[string]any)
	if !ok {
		t.Fatalf("hooks = %#v", root["hooks"])
	}
	if _, ok := hooks["PreToolUse"]; !ok {
		t.Fatal("an existing event key was replaced")
	}
	groups := array(t, hooks["WorktreeCreate"], "WorktreeCreate")
	if len(groups) != 1 {
		t.Fatalf("WorktreeCreate = %v", groups)
	}
	group, ok := groups[0].(map[string]any)
	if !ok {
		t.Fatalf("group = %#v", groups[0])
	}
	handlers := array(t, group["hooks"], "hooks")
	handler, ok := handlers[0].(map[string]any)
	if !ok {
		t.Fatalf("handler = %#v", handlers[0])
	}
	command, _ := handler["command"].(string)
	if handler["type"] != "command" || !strings.Contains(command, "berth adopt --setup") {
		t.Fatalf("handler = %#v", handler)
	}
	// A non-zero exit from WorktreeCreate aborts Claude's worktree creation, so
	// the installed command must swallow the failure itself.
	if !strings.Contains(command, "exit 0") {
		t.Fatalf("command %q does not guarantee exit 0", command)
	}

	changed, err = writeClaudeHook(repo)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("second merge rewrote the file")
	}
	if again := readFile(t, path); again != first {
		t.Fatal("second merge changed the file")
	}
}

func TestHookInstalledDetection(t *testing.T) {
	repo := t.TempDir()
	h, err := skill.LookupHarness("cursor")
	if err != nil {
		t.Fatal(err)
	}
	if skill.HookInstalled(repo, h) {
		t.Fatal("no hook file exists yet")
	}
	if _, err := writeCursorHook(repo); err != nil {
		t.Fatal(err)
	}
	if !skill.HookInstalled(repo, h) {
		t.Fatal("the hook berth just wrote is not detected")
	}
}

// TestInstallCommandsInRepository drives the app-level wiring the CLI uses.
func TestInstallCommandsInRepository(t *testing.T) {
	t.Setenv("BERTH_HOME", t.TempDir())
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	repo := initRepo(t)
	orig, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Chdir(repo); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chdir(orig) })

	ctx := context.Background()
	a, err := Open(ctx)
	if err != nil {
		t.Fatal(err)
	}
	defer a.Close()

	results, err := a.SkillInstall(ctx, skill.Options{Agents: []string{"claude,cline"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("results = %+v", results)
	}
	for _, r := range results {
		if r.Scope != skill.ScopeProject || !r.Changed {
			t.Fatalf("result = %+v", r)
		}
	}
	for _, rel := range []string{".claude/skills/berth/SKILL.md", ".cline/skills/berth/SKILL.md", ".claude/skills/berth/references/recovery.md"} {
		if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("%s: %v", rel, err)
		}
	}
	if _, err := os.Stat(filepath.Join(repo, ".agents")); !os.IsNotExist(err) {
		t.Fatal("--agent claude,cline must not write the shared path")
	}
	if entries, err := os.ReadDir(home); err != nil || len(entries) != 0 {
		t.Fatalf("project scope touched the home directory: %v %v", entries, err)
	}

	again, err := a.SkillInstall(ctx, skill.Options{Agents: []string{"claude,cline"}})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range again {
		if r.Changed {
			t.Fatalf("second install rewrote %s", r.Path)
		}
	}

	// User scope writes under the fake home only.
	if _, err := a.SkillInstall(ctx, skill.Options{Scope: skill.ScopeUser}); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(home, ".agents", "skills", "berth", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(repo, ".agents")); !os.IsNotExist(err) {
		t.Fatal("user scope wrote inside the repository")
	}

	hooks, err := a.HookInstall(ctx, skill.Options{})
	if err != nil {
		t.Fatal(err)
	}
	if len(hooks) != 2 || hooks[0].Agent != "cursor" || hooks[1].Agent != "windsurf" {
		t.Fatalf("hooks = %+v", hooks)
	}
	for _, rel := range []string{".cursor/worktrees.json", ".windsurf/hooks.json"} {
		if _, err := os.Stat(filepath.Join(repo, filepath.FromSlash(rel))); err != nil {
			t.Fatalf("%s: %v", rel, err)
		}
	}
	if _, err := os.Stat(filepath.Join(repo, ".claude", "settings.json")); !os.IsNotExist(err) {
		t.Fatal("the opt-in Claude hook was installed by default")
	}

	// Idempotency: the second run must report no change and rewrite nothing.
	before := readFile(t, filepath.Join(repo, ".cursor", "worktrees.json"))
	againHooks, err := a.HookInstall(ctx, skill.Options{})
	if err != nil {
		t.Fatal(err)
	}
	for _, r := range againHooks {
		if r.Changed {
			t.Fatalf("second hook install rewrote %s", r.Path)
		}
	}
	if after := readFile(t, filepath.Join(repo, ".cursor", "worktrees.json")); after != before {
		t.Fatal("second hook install changed the file")
	}

	// Explicit opt-in, and user scope refused before anything is written.
	claude, err := a.HookInstall(ctx, skill.Options{Agents: []string{"claude"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(claude) != 1 || claude[0].Agent != "claude" || !claude[0].Changed {
		t.Fatalf("claude hook = %+v", claude)
	}
	hookFiles := []string{".cursor/worktrees.json", ".windsurf/hooks.json", ".claude/settings.json"}
	beforeUser := map[string]string{}
	for _, rel := range hookFiles {
		beforeUser[rel] = readFile(t, filepath.Join(repo, filepath.FromSlash(rel)))
	}
	if _, err := a.HookInstall(ctx, skill.Options{Scope: skill.ScopeUser}); err != nil {
		t.Fatal(err)
	}
	for _, rel := range hookFiles {
		if got := readFile(t, filepath.Join(repo, filepath.FromSlash(rel))); got != beforeUser[rel] {
			t.Fatalf("a user-scope hook install wrote %s", rel)
		}
	}

	if _, err := a.HookInstall(ctx, skill.Options{Agents: []string{"zed"}}); err == nil {
		t.Fatal("a harness without a hook must be an error")
	}
}
