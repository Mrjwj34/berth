package skill

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// Scope selects the root an install writes into. There is no third scope: a
// harness reads a skill either from the repository it was started in or from
// the user profile.
const (
	ScopeProject = "project"
	ScopeUser    = "user"
)

// SkillFileName is the file every harness reads inside a skill directory.
const SkillFileName = "SKILL.md"

// SharedProjectPath and SharedUserPath are the .agents/skills convention: the
// one copy of the skill that twelve harnesses read without a per-harness copy
// (see the Covers list on the shared registry entry).
const (
	SharedProjectPath = ".agents/skills/berth"
	SharedUserPath    = ".agents/skills/berth"
)

// sharedName is the selector for the shared .agents/skills target.
const sharedName = "shared"

// HookMarker is the text berth looks for when deciding whether a harness's hook
// configuration already carries a berth entry. A hook file that mentions it is
// left alone, so a repeated install never duplicates an entry a user edited.
const HookMarker = "berth adopt"

// Harness is one install target: a harness that needs its own copy of the
// skill, or the shared .agents/skills convention that several harnesses read.
//
// Paths are relative to the repository root (ProjectPath) or to the home
// directory (UserPath), always slash-separated and never absolute, so the
// registry cannot address anything outside the scope it is installed into.
type Harness struct {
	// Name is the stable selector accepted by --agent.
	Name string
	// Display is the harness name as its own documentation writes it.
	Display string
	// Shared records that this target is the shared .agents/skills convention,
	// or a harness served by it.
	Shared bool
	// ProjectPath and UserPath are the skill directories for each scope.
	ProjectPath string
	UserPath    string
	// Hook is the event berth installs into HookFile ("" when berth installs no
	// hook for this harness), and HookFile is the repository-relative config
	// file that carries it. HookOptIn marks a hook that `hook install all`
	// deliberately skips because a bad entry can abort worktree creation.
	Hook      string
	HookFile  string
	HookOptIn bool
	// Covers lists the harnesses a shared target serves.
	Covers []string
	// Note is one line of context for `berth agents`.
	Note string
}

// The registry is a plain ordered slice: `berth agents`, --all and the error
// messages all read it in this order, so the output is deterministic. The
// shared convention comes first because it is what a bare install writes, then
// the harnesses that need their own copy, then every harness the shared
// convention covers.
var harnesses = []Harness{
	{
		Name:        sharedName,
		Display:     "Shared .agents/skills convention",
		Shared:      true,
		ProjectPath: SharedProjectPath,
		UserPath:    SharedUserPath,
		Covers: []string{
			"codex", "cursor", "copilot", "gemini", "opencode", "windsurf",
			"roo", "kilo", "zed", "junie", "antigravity", "pi",
		},
		Note: "One canonical copy of the skill; this is what berth installs with no --agent flag.",
	},
	{
		Name:        "claude",
		Display:     "Claude Code",
		ProjectPath: ".claude/skills/berth",
		UserPath:    ".claude/skills/berth",
		Hook:        "WorktreeCreate",
		HookFile:    ".claude/settings.json",
		HookOptIn:   true,
		Note:        "Does not read .agents/skills or AGENTS.md, so it needs its own copy.",
	},
	{
		Name:        "cline",
		Display:     "Cline",
		ProjectPath: ".cline/skills/berth",
		UserPath:    ".cline/skills/berth",
		Note:        "Does not read .agents/skills; reads .cline/skills, .clinerules/skills and .claude/skills.",
	},
	{
		Name:        "codex",
		Display:     "OpenAI Codex CLI",
		Shared:      true,
		ProjectPath: SharedProjectPath,
		UserPath:    SharedUserPath,
		Note:        "Scans .agents/skills from the working directory up to the repository root; no worktree setup hook is documented.",
	},
	{
		Name:        "cursor",
		Display:     "Cursor",
		Shared:      true,
		ProjectPath: SharedProjectPath,
		UserPath:    SharedUserPath,
		Hook:        "setup-worktree",
		HookFile:    ".cursor/worktrees.json",
		Note:        "Runs .cursor/worktrees.json setup commands in every worktree it creates in the Agents Window, the IDE and the CLI.",
	},
	{
		Name:        "copilot",
		Display:     "GitHub Copilot",
		Shared:      true,
		ProjectPath: SharedProjectPath,
		UserPath:    SharedUserPath,
		Note:        "VS Code agent mode reads .agents/skills; the cloud agent runs on Actions runners, where berth's worktree model does not apply.",
	},
	{
		Name:        "gemini",
		Display:     "Gemini CLI",
		Shared:      true,
		ProjectPath: SharedProjectPath,
		UserPath:    SharedUserPath,
		Note:        "The .agents/skills alias wins inside a tier; its experimental worktrees have no documented setup hook.",
	},
	{
		Name:        "opencode",
		Display:     "opencode",
		Shared:      true,
		ProjectPath: SharedProjectPath,
		UserPath:    SharedUserPath,
		Note:        "Reads .agents/skills; its hooks are TypeScript plugins, which berth does not install.",
	},
	{
		Name:        "windsurf",
		Display:     "Windsurf / Devin Desktop",
		Shared:      true,
		ProjectPath: SharedProjectPath,
		UserPath:    SharedUserPath,
		Hook:        "post_setup_worktree",
		HookFile:    ".windsurf/hooks.json",
		Note:        "post_setup_worktree runs inside each new worktree with $ROOT_WORKSPACE_PATH set.",
	},
	{
		Name:        "roo",
		Display:     "Roo Code",
		Shared:      true,
		ProjectPath: SharedProjectPath,
		UserPath:    SharedUserPath,
		Note:        "Reads .agents/skills, but the extension was shut down on 2026-05-15; berth installs no hook for it.",
	},
	{
		Name:        "kilo",
		Display:     "Kilo Code",
		Shared:      true,
		ProjectPath: SharedProjectPath,
		UserPath:    SharedUserPath,
		Note:        "Loads .agents/skills by default; its hooks are TypeScript plugins, which berth does not install.",
	},
	{
		Name:        "zed",
		Display:     "Zed",
		Shared:      true,
		ProjectPath: SharedProjectPath,
		UserPath:    SharedUserPath,
		Note:        ".agents/skills is Zed's only skills root; project skills load only from trusted worktrees.",
	},
	{
		Name:        "junie",
		Display:     "JetBrains Junie",
		Shared:      true,
		ProjectPath: SharedProjectPath,
		UserPath:    SharedUserPath,
		Note:        "Reads .agents/skills in a trusted project; it has no hook surface at all, so berth can never adopt its worktrees automatically.",
	},
	{
		Name:        "antigravity",
		Display:     "Google Antigravity",
		Shared:      true,
		ProjectPath: SharedProjectPath,
		UserPath:    SharedUserPath,
		Note:        "Defaults to .agents/skills; its worktree mode has no documented repository-side setup hook.",
	},
	{
		Name:        "pi",
		Display:     "pi",
		Shared:      true,
		ProjectPath: SharedProjectPath,
		UserPath:    SharedUserPath,
		Note:        "Reads .agents/skills in the working directory and its ancestors once the project is trusted; worktrees are extension territory only.",
	},
}

// Harnesses returns the registry in presentation order.
func Harnesses() []Harness { return append([]Harness(nil), harnesses...) }

// LookupHarness resolves an --agent value.
func LookupHarness(name string) (Harness, error) {
	for _, h := range harnesses {
		if h.Name == name {
			return h, nil
		}
	}
	return Harness{}, fmt.Errorf("unknown agent %q. Valid names: %s; or pass --all", name, strings.Join(HarnessNames(), ", "))
}

// HarnessNames lists every selector --agent accepts, in registry order.
func HarnessNames() []string { return names(harnesses) }

// HookNames lists the harnesses berth can install a worktree hook for.
func HookNames() []string { return names(withHook(harnesses)) }

func names(list []Harness) []string {
	out := make([]string, 0, len(list))
	for _, h := range list {
		out = append(out, h.Name)
	}
	return out
}

func withHook(list []Harness) []Harness {
	out := []Harness{}
	for _, h := range list {
		if h.HookFile != "" {
			out = append(out, h)
		}
	}
	return out
}

// defaultHooks returns the hooks a bare `berth hook install` and `all` write:
// every hook that is not opt-in.
func defaultHooks() []Harness {
	out := []Harness{}
	for _, h := range withHook(harnesses) {
		if !h.HookOptIn {
			out = append(out, h)
		}
	}
	return out
}

// Options selects what an install command writes.
type Options struct {
	// Agents holds repeated --agent values; each may also be a comma-separated
	// list. Empty means the default: the shared skill target for skills, every
	// default hook for hooks.
	Agents []string
	// All selects every supported harness.
	All bool
	// Scope is ScopeProject (default) or ScopeUser.
	Scope string
}

// Normalize validates --agent and --scope once, so callers and tests share one
// definition of what those flags accept.
func (o Options) Normalize() (Options, error) {
	scope, err := ParseScope(o.Scope)
	if err != nil {
		return Options{}, err
	}
	agents, err := SplitAgents(o.Agents)
	if err != nil {
		return Options{}, err
	}
	o.Scope, o.Agents = scope, agents
	return o, nil
}

// ParseScope accepts the empty string as the project default.
func ParseScope(value string) (string, error) {
	switch value {
	case "":
		return ScopeProject, nil
	case ScopeProject, ScopeUser:
		return value, nil
	default:
		return "", fmt.Errorf("unknown scope %q. Use project (default) or user", value)
	}
}

// SplitAgents accepts repeated --agent flags and comma-separated lists in any
// mix. An empty name is an error rather than a silent widening of the install.
func SplitAgents(values []string) ([]string, error) {
	out := []string{}
	for _, value := range values {
		for _, part := range strings.Split(value, ",") {
			name := strings.TrimSpace(part)
			if name == "" {
				return nil, fmt.Errorf("empty agent name in --agent %q; name a harness or pass --all", value)
			}
			out = append(out, name)
		}
	}
	return out, nil
}

// Target is one skill directory an install writes into.
type Target struct {
	Harness Harness
	Scope   string
	// Dir is absolute; Root is the absolute scope root it was derived from.
	Dir  string
	Root string
}

// Rel is the target directory relative to its scope root, slash-separated so
// output is identical on Windows and on Unix.
func (t Target) Rel() string {
	rel, err := filepath.Rel(t.Root, t.Dir)
	if err != nil {
		return filepath.ToSlash(t.Dir)
	}
	return filepath.ToSlash(rel)
}

// SkillTargets resolves the skill directories a command should write, in
// registry order and without duplicates: selecting two names that share the
// .agents/skills convention still writes that copy once.
func (o Options) SkillTargets(repoRoot, homeDir string) ([]Target, error) {
	list, err := o.skillHarnesses()
	if err != nil {
		return nil, err
	}
	out := []Target{}
	seen := map[string]bool{}
	for _, h := range list {
		dir, err := DirFor(h, o.Scope, repoRoot, homeDir)
		if err != nil {
			return nil, err
		}
		if seen[dir] {
			continue
		}
		seen[dir] = true
		root := repoRoot
		if o.Scope == ScopeUser {
			root = homeDir
		}
		out = append(out, Target{Harness: h, Scope: o.Scope, Dir: dir, Root: root})
	}
	return out, nil
}

func (o Options) skillHarnesses() ([]Harness, error) {
	if o.All {
		return Harnesses(), nil
	}
	if len(o.Agents) == 0 {
		h, err := LookupHarness(sharedName)
		if err != nil {
			return nil, err
		}
		return []Harness{h}, nil
	}
	out := []Harness{}
	for _, name := range o.Agents {
		h, err := LookupHarness(name)
		if err != nil {
			return nil, err
		}
		out = append(out, h)
	}
	return out, nil
}

// HookTargets resolves the harnesses whose worktree hook a command should
// merge. A named harness must actually have a hook: silently doing nothing for
// a harness berth cannot hook would read as a successful install.
func (o Options) HookTargets() ([]Harness, error) {
	out := []Harness{}
	if o.All || len(o.Agents) == 0 {
		out = append(out, defaultHooks()...)
	}
	for _, name := range o.Agents {
		h, err := LookupHarness(name)
		if err != nil {
			return nil, err
		}
		if h.HookFile == "" {
			return nil, fmt.Errorf("harness %q has no worktree hook berth installs. Harnesses with a hook: %s; install the skill instead with berth skill install --agent %s", name, strings.Join(HookNames(), ", "), name)
		}
		out = append(out, h)
	}
	seen := map[string]bool{}
	uniq := out[:0]
	for _, h := range out {
		if seen[h.Name] {
			continue
		}
		seen[h.Name] = true
		uniq = append(uniq, h)
	}
	return uniq, nil
}

// DirFor resolves the directory a harness reads its skill from at this scope.
// The result is always the scope root joined with a registry-relative path, and
// the join is checked to stay inside that root: project scope can never write
// outside the repository and user scope can never write inside it.
func DirFor(h Harness, scope, repoRoot, homeDir string) (string, error) {
	rel, root, err := h.scopePath(scope, repoRoot, homeDir)
	if err != nil {
		return "", err
	}
	return contained(root, rel)
}

func (h Harness) scopePath(scope, repoRoot, homeDir string) (string, string, error) {
	switch scope {
	case ScopeProject:
		if repoRoot == "" {
			return "", "", fmt.Errorf("harness %s: project scope needs a repository root", h.Name)
		}
		return h.ProjectPath, repoRoot, nil
	case ScopeUser:
		if homeDir == "" {
			return "", "", fmt.Errorf("harness %s: user scope needs a home directory", h.Name)
		}
		return h.UserPath, homeDir, nil
	default:
		return "", "", fmt.Errorf("unknown scope %q. Use project (default) or user", scope)
	}
}

// contained joins a registry-relative path onto root and refuses anything that
// would escape it.
func contained(root, rel string) (string, error) {
	joined := filepath.Join(root, filepath.FromSlash(rel))
	inside, err := filepath.Rel(root, joined)
	if err != nil {
		return "", fmt.Errorf("resolve %s: %w", rel, err)
	}
	if inside == ".." || strings.HasPrefix(inside, ".."+string(filepath.Separator)) || filepath.IsAbs(inside) {
		return "", fmt.Errorf("%s escapes its scope root", rel)
	}
	return joined, nil
}

// HomeDir resolves the user-scope root. berth never expands a literal "~":
// Windows shells do not, and os.UserHomeDir already reads USERPROFILE there and
// HOME on Unix.
func HomeDir() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve the home directory for user scope: %w", err)
	}
	if home == "" {
		return "", fmt.Errorf("resolve the home directory for user scope: set HOME or USERPROFILE")
	}
	return home, nil
}

// AgentRow is one harness in the `berth agents` report. Field order here is the
// stable field order of the JSON output.
type AgentRow struct {
	Name        string `json:"name"`
	Display     string `json:"display"`
	ReadsShared bool   `json:"reads_shared"`
	ProjectPath string `json:"project_path"`
	UserPath    string `json:"user_path"`
	// Hook is the harness event berth installs into HookFile; empty means berth
	// installs no hook for this harness.
	Hook      string `json:"hook"`
	HookFile  string `json:"hook_file"`
	HookOptIn bool   `json:"hook_opt_in"`

	InstalledProject bool `json:"installed_project"`
	InstalledUser    bool `json:"installed_user"`
	// HookInstalled reports a berth entry already present in HookFile. It is
	// only meaningful when Hook is non-empty.
	HookInstalled bool `json:"hook_installed"`

	Covers []string `json:"covers,omitempty"`
	Note   string   `json:"note"`
}

// Report is what `berth agents` prints: the registry plus what is already
// installed in this repository and in the user profile.
type Report struct {
	Repo   string     `json:"repo"`
	Home   string     `json:"home"`
	Agents []AgentRow `json:"agents"`
}

// BuildReport builds the discovery report. It only reads the filesystem:
// nothing here writes, so `berth agents` is safe to run anywhere.
func BuildReport(repoRoot, homeDir string) Report {
	rep := Report{Repo: repoRoot, Home: homeDir, Agents: []AgentRow{}}
	for _, h := range harnesses {
		row := AgentRow{
			Name:        h.Name,
			Display:     h.Display,
			ReadsShared: h.Shared,
			ProjectPath: h.ProjectPath,
			UserPath:    h.UserPath,
			Hook:        h.Hook,
			HookFile:    h.HookFile,
			HookOptIn:   h.HookOptIn,
			Covers:      h.Covers,
			Note:        h.Note,
		}
		if dir, err := DirFor(h, ScopeProject, repoRoot, homeDir); err == nil {
			row.InstalledProject = installed(dir)
		}
		if dir, err := DirFor(h, ScopeUser, repoRoot, homeDir); err == nil {
			row.InstalledUser = installed(dir)
		}
		row.HookInstalled = HookInstalled(repoRoot, h)
		rep.Agents = append(rep.Agents, row)
	}
	return rep
}

// installed reports whether a skill copy is present in dir.
func installed(dir string) bool {
	st, err := os.Stat(filepath.Join(dir, SkillFileName))
	return err == nil && st.Mode().IsRegular()
}

// HookInstalled reports whether a harness's hook file already carries a berth
// entry. A file berth cannot read counts as not installed; the merge then
// reports the real problem when it tries to write.
func HookInstalled(repoRoot string, h Harness) bool {
	if h.HookFile == "" || repoRoot == "" {
		return false
	}
	data, err := os.ReadFile(filepath.Join(repoRoot, filepath.FromSlash(h.HookFile)))
	if err != nil {
		return false
	}
	return bytes.Contains(data, []byte(HookMarker))
}
