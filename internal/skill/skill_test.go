package skill

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallWritesSkillUnderAgents(t *testing.T) {
	root := t.TempDir()
	if err := Install(root); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(SkillPath(root))
	if err != nil {
		t.Fatal(err)
	}
	if len(data) < 100 {
		t.Fatalf("%s too small", SkillPath(root))
	}
	for _, stale := range []string{
		".claude/skills/berth/SKILL.md",
		".cursor/skills/berth/SKILL.md",
		".berth/hooks",
	} {
		if _, err := os.Stat(filepath.Join(root, filepath.FromSlash(stale))); !os.IsNotExist(err) {
			t.Fatalf("a bare install must not write outside .agents: %s", stale)
		}
	}
}

// TestDefaultSelectionIsTheSharedPath pins the no-flags behaviour: one target,
// the shared .agents/skills copy.
func TestDefaultSelectionIsTheSharedPath(t *testing.T) {
	repo, home := t.TempDir(), t.TempDir()
	opts, err := Options{}.Normalize()
	if err != nil {
		t.Fatal(err)
	}
	if opts.Scope != ScopeProject {
		t.Fatalf("default scope = %q", opts.Scope)
	}
	targets, err := opts.SkillTargets(repo, home)
	if err != nil {
		t.Fatal(err)
	}
	if len(targets) != 1 {
		t.Fatalf("targets = %d, want 1", len(targets))
	}
	if got := targets[0].Rel(); got != ".agents/skills/berth" {
		t.Fatalf("default target = %s", got)
	}
	if _, err := InstallInto(targets[0].Dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(repo, ".agents", "skills", "berth", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	if entries, err := os.ReadDir(home); err != nil || len(entries) != 0 {
		t.Fatalf("default install touched the home directory: %v %v", entries, err)
	}
}

func TestSkillTargetsPerAgent(t *testing.T) {
	repo, home := t.TempDir(), t.TempDir()
	cases := []struct {
		name   string
		agents []string
		all    bool
		scope  string
		want   []string
	}{
		{name: "claude", agents: []string{"claude"}, want: []string{".claude/skills/berth"}},
		{name: "cline", agents: []string{"cline"}, want: []string{".cline/skills/berth"}},
		{name: "comma separated", agents: []string{"claude,cline"}, want: []string{".claude/skills/berth", ".cline/skills/berth"}},
		{name: "repeated flags", agents: []string{"cline", "claude"}, want: []string{".cline/skills/berth", ".claude/skills/berth"}},
		{name: "cursor uses the shared copy", agents: []string{"cursor"}, want: []string{".agents/skills/berth"}},
		{name: "two names one shared copy", agents: []string{"cursor,zed"}, want: []string{".agents/skills/berth"}},
		{name: "explicit shared", agents: []string{"shared"}, want: []string{".agents/skills/berth"}},
		{name: "all", all: true, want: []string{".agents/skills/berth", ".claude/skills/berth", ".cline/skills/berth"}},
		{name: "all at user scope", all: true, scope: ScopeUser, want: []string{".agents/skills/berth", ".claude/skills/berth", ".cline/skills/berth"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opts, err := Options{Agents: tc.agents, All: tc.all, Scope: tc.scope}.Normalize()
			if err != nil {
				t.Fatal(err)
			}
			targets, err := opts.SkillTargets(repo, home)
			if err != nil {
				t.Fatal(err)
			}
			got := make([]string, 0, len(targets))
			for _, target := range targets {
				got = append(got, target.Rel())
			}
			if strings.Join(got, ",") != strings.Join(tc.want, ",") {
				t.Fatalf("targets = %v, want %v", got, tc.want)
			}
			for _, target := range targets {
				if filepath.ToSlash(target.Rel()) != target.Harness.ProjectPath {
					t.Fatalf("target %s resolved to %s", target.Harness.Name, target.Rel())
				}
			}
		})
	}
}

// TestScopeNeverEscapesItsRoot proves both invariants: project scope never
// writes outside the repository, user scope never writes inside it.
func TestScopeNeverEscapesItsRoot(t *testing.T) {
	for _, scope := range []string{ScopeProject, ScopeUser} {
		t.Run(scope, func(t *testing.T) {
			repo, home := t.TempDir(), t.TempDir()
			opts, err := Options{All: true, Scope: scope}.Normalize()
			if err != nil {
				t.Fatal(err)
			}
			targets, err := opts.SkillTargets(repo, home)
			if err != nil {
				t.Fatal(err)
			}
			if len(targets) == 0 {
				t.Fatal("no targets")
			}
			for _, target := range targets {
				changed, err := InstallInto(target.Dir)
				if err != nil {
					t.Fatal(err)
				}
				if !changed {
					t.Fatalf("%s: fresh install reported no change", target.Dir)
				}
				inside := repo
				if target.Scope == ScopeUser {
					inside = home
				}
				if rel, err := filepath.Rel(inside, target.Dir); err != nil || strings.HasPrefix(rel, "..") {
					t.Fatalf("%s scope wrote outside %s: %s", scope, inside, target.Dir)
				}
			}
			outside := repo
			if scope == ScopeProject {
				outside = home
			}
			if entries, err := os.ReadDir(outside); err != nil || len(entries) != 0 {
				t.Fatalf("%s scope wrote outside its root: %v %v", scope, entries, err)
			}
			inside := home
			if scope == ScopeProject {
				inside = repo
			}
			if countFiles(t, inside) == 0 {
				t.Fatalf("%s scope wrote nothing", scope)
			}
		})
	}
}

func TestUnknownAgentIsActionable(t *testing.T) {
	for _, name := range []string{"copilot-cloud", "claude-code", "roo-code"} {
		_, err := Options{Agents: []string{name}}.SkillTargets(t.TempDir(), t.TempDir())
		if err == nil {
			t.Fatalf("%s: expected an error", name)
		}
		for _, want := range []string{name, "claude", "cline", "shared", "--all"} {
			if !strings.Contains(err.Error(), want) {
				t.Fatalf("error %q does not mention %q", err, want)
			}
		}
	}
	if _, err := (Options{Agents: []string{"cursor,"}}).Normalize(); err == nil {
		t.Fatal("expected an error for an empty agent name")
	}
	if _, err := ParseScope("global"); err == nil {
		t.Fatal("expected an error for an unknown scope")
	}
}

func TestHookTargets(t *testing.T) {
	cases := []struct {
		name    string
		agents  []string
		all     bool
		want    []string
		wantErr string
	}{
		{name: "default installs every default hook", want: []string{"cursor", "windsurf"}},
		{name: "all skips the opt-in hook", all: true, want: []string{"cursor", "windsurf"}},
		{name: "cursor", agents: []string{"cursor"}, want: []string{"cursor"}},
		{name: "windsurf", agents: []string{"windsurf"}, want: []string{"windsurf"}},
		{name: "claude is opt-in but selectable", agents: []string{"claude"}, want: []string{"claude"}},
		{name: "claude with all", agents: []string{"claude"}, all: true, want: []string{"cursor", "windsurf", "claude"}},
		{name: "duplicates collapse", agents: []string{"cursor,cursor"}, want: []string{"cursor"}},
		{name: "harness without a hook", agents: []string{"zed"}, wantErr: "no worktree hook"},
		{name: "unknown agent", agents: []string{"zedd"}, wantErr: "unknown agent"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			opts, err := Options{Agents: tc.agents, All: tc.all}.Normalize()
			if err != nil {
				t.Fatal(err)
			}
			got, err := opts.HookTargets()
			if tc.wantErr != "" {
				if err == nil || !strings.Contains(err.Error(), tc.wantErr) {
					t.Fatalf("error = %v, want %q", err, tc.wantErr)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			names := make([]string, 0, len(got))
			for _, h := range got {
				names = append(names, h.Name)
			}
			if strings.Join(names, ",") != strings.Join(tc.want, ",") {
				t.Fatalf("hooks = %v, want %v", names, tc.want)
			}
		})
	}
}

func TestInstallIntoIsIdempotent(t *testing.T) {
	dir := filepath.Join(t.TempDir(), ".agents", "skills", "berth")
	changed, err := InstallInto(dir)
	if err != nil {
		t.Fatal(err)
	}
	if !changed {
		t.Fatal("first install reported no change")
	}
	before := snapshot(t, dir)
	for _, ref := range []string{"references/berth-yaml.md", "references/recovery.md"} {
		if _, err := os.Stat(filepath.Join(dir, filepath.FromSlash(ref))); err != nil {
			t.Fatalf("pointer %s does not resolve: %v", ref, err)
		}
	}
	changed, err = InstallInto(dir)
	if err != nil {
		t.Fatal(err)
	}
	if changed {
		t.Fatal("second install rewrote an up-to-date skill")
	}
	after := snapshot(t, dir)
	if len(before) != len(after) {
		t.Fatalf("file set changed: %d then %d", len(before), len(after))
	}
	for name, data := range before {
		if string(after[name]) != string(data) {
			t.Fatalf("%s is not byte-identical after a second install", name)
		}
	}
}

func TestRegistryInvariants(t *testing.T) {
	list := Harnesses()
	if len(list) == 0 || list[0].Name != sharedName {
		t.Fatalf("the shared convention must come first, got %v", HarnessNames())
	}
	seen := map[string]bool{}
	for _, h := range list {
		if h.Name == "" || h.Display == "" || h.Note == "" {
			t.Fatalf("incomplete entry: %+v", h)
		}
		if seen[h.Name] {
			t.Fatalf("duplicate harness name %q", h.Name)
		}
		seen[h.Name] = true
		for _, path := range []string{h.ProjectPath, h.UserPath, h.HookFile} {
			if path == "" {
				continue
			}
			if filepath.IsAbs(path) || strings.Contains(path, `\`) || strings.Contains(path, "..") || strings.HasPrefix(path, "/") {
				t.Fatalf("%s: path %q must be relative and slash-separated", h.Name, path)
			}
		}
		if h.HookFile != "" && h.Hook == "" {
			t.Fatalf("%s: hook file without a hook event", h.Name)
		}
	}
	shared, err := LookupHarness(sharedName)
	if err != nil {
		t.Fatal(err)
	}
	if len(shared.Covers) != 12 {
		t.Fatalf("the shared convention covers %d harnesses, want 12: %v", len(shared.Covers), shared.Covers)
	}
	for _, name := range shared.Covers {
		h, err := LookupHarness(name)
		if err != nil {
			t.Fatal(err)
		}
		if !h.Shared || h.ProjectPath != SharedProjectPath || h.UserPath != SharedUserPath {
			t.Fatalf("covered harness %s does not use the shared convention", name)
		}
	}
}

func TestReportIsDeterministic(t *testing.T) {
	repo, home := t.TempDir(), t.TempDir()
	first, err := json.MarshalIndent(BuildReport(repo, home), "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	second, err := json.MarshalIndent(BuildReport(repo, home), "", "  ")
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatal("report is not deterministic")
	}
	order := []string{`"repo"`, `"home"`, `"agents"`, `"name"`, `"display"`, `"reads_shared"`, `"project_path"`, `"user_path"`,
		`"hook"`, `"hook_file"`, `"hook_opt_in"`, `"installed_project"`, `"installed_user"`, `"hook_installed"`, `"covers"`, `"note"`}
	at := 0
	for _, field := range order {
		found := strings.Index(string(first), field)
		if found < 0 {
			t.Fatalf("field %s missing from the JSON report", field)
		}
		if found < at {
			t.Fatalf("field %s is out of the documented order", field)
		}
		at = found
	}
	if err := Install(repo); err != nil {
		t.Fatal(err)
	}
	rep := BuildReport(repo, home)
	if !rep.Agents[0].InstalledProject {
		t.Fatal("the shared skill is installed but the report says otherwise")
	}
	if rep.Agents[0].InstalledUser {
		t.Fatal("nothing was installed at user scope")
	}
	for _, row := range rep.Agents {
		if row.Name == "claude" || row.Name == "cline" {
			if row.InstalledProject {
				t.Fatalf("%s: false positive for a path berth never wrote", row.Name)
			}
		}
	}
}

func TestHomeDirFollowsTheEnvironment(t *testing.T) {
	fake := t.TempDir()
	t.Setenv("HOME", fake)
	t.Setenv("USERPROFILE", fake)
	got, err := HomeDir()
	if err != nil {
		t.Fatal(err)
	}
	if got != fake {
		t.Fatalf("HomeDir() = %q, want %q", got, fake)
	}
}

func countFiles(t *testing.T, root string) int {
	t.Helper()
	count := 0
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			count++
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return count
}

func snapshot(t *testing.T, root string) map[string][]byte {
	t.Helper()
	out := map[string][]byte{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		out[filepath.ToSlash(rel)] = data
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}
