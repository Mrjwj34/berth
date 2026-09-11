# Agent harness matrix — skills, instructions, hooks, worktrees

Reference for implementing berth's multi-harness artifact installer (`berth init` / `berth hook install`).
Compiled 2026-09-12 from first-party documentation, first-party repository sources, and commands executed
on this machine. Every path and command below is followed by the URL it came from, or is marked
**UNVERIFIED**.

## How to read this document

| Marker | Meaning |
| --- | --- |
| `[source: URL]` | The fact comes from that page (or from the raw markdown of that page). |
| **(exec)** | The fact comes from a command run in this session; the observed output is quoted. |
| **UNVERIFIED** | Could not confirm from a first-party source. Do not implement against it without checking. |
| **DISCREPANCY** | Two first-party sources disagree. Both are quoted. |

Confidence rule used throughout: a harness's *own* documentation or source tree is first-party. A
third-party installer's table describing another product (for example the `skills` CLI's supported-agent
table) is treated as a secondary source for that product and is labelled as such.

---

## 1. Summary table — discovery and worktrees

"`.agents/skills`?" = the harness documents reading the shared `.agents/skills/<name>/SKILL.md`
convention. Details and citations in §2 and §3.

| Harness | Reads `.agents/skills`? | Project skills | Global (user) skills | Always-on instructions | Creates Git worktrees? |
| --- | --- | --- | --- | --- | --- |
| **Claude Code** | **No** | `.claude/skills/<name>/SKILL.md` | `~/.claude/skills/<name>/SKILL.md` | `./CLAUDE.md`, `./.claude/CLAUDE.md`, `.claude/rules/*.md`; user `~/.claude/CLAUDE.md`, `~/.claude/rules/` | **Yes** — `.claude/worktrees/<name>/` |
| **OpenAI Codex CLI** | **Yes** | `.agents/skills/` (cwd → repo root) | `~/.agents/skills/` | `AGENTS.override.md` then `AGENTS.md`; user `~/.codex/AGENTS.md` | Not documented (UNVERIFIED) |
| **Cursor** | **Yes** | `.agents/skills/`, `.cursor/skills/` | `~/.agents/skills/`, `~/.cursor/skills/` | `.cursor/rules/*.mdc`, `AGENTS.md`, `CLAUDE.md`; user `~/.cursor/rules` | **Yes** — `~/.cursor/worktrees/<repo>/<name>`; `.cursor/worktrees.json` |
| **GitHub Copilot** (VS Code agent mode) | **Yes** | `.github/skills/`, `.claude/skills/`, `.agents/skills/` | `~/.copilot/skills/`, `~/.claude/skills/`, `~/.agents/skills/` | `.github/copilot-instructions.md`, `.github/instructions/*.instructions.md`, `.claude/rules/`, `AGENTS.md`, `CLAUDE.md`; user `~/.copilot/instructions` | No (cloud agent uses Actions runners) |
| **GitHub Copilot cloud agent** | **Yes** (same roots) | same as above | same as above | same; environment file `.github/workflows/copilot-setup-steps.yml` | Not documented (UNVERIFIED) |
| **Gemini CLI** | **Yes** (alias that *wins*) | `.agents/skills/` or `.gemini/skills/` | `~/.agents/skills/` or `~/.gemini/skills/` | `GEMINI.md` + parents; user `~/.gemini/GEMINI.md` | **Yes** — `.gemini/worktrees/<name>/` (experimental) |
| **opencode** | **Yes** | `.opencode/skills/`, `.claude/skills/`, `.agents/skills/` | `~/.config/opencode/skills/`, `~/.claude/skills/`, `~/.agents/skills/` | `AGENTS.md` (`CLAUDE.md` fallback); user `~/.config/opencode/AGENTS.md` | No (only a `worktree` string in plugin context) |
| **Windsurf / Devin Desktop** | **Yes** | `.windsurf/skills/`, `.agents/skills/` | `~/.codeium/windsurf/skills/`, `~/.agents/skills/` | `.devin/rules/*.md` (preferred), `.windsurf/rules/*.md`, `.windsurfrules`, `AGENTS.md`; user `~/.codeium/windsurf/memories/global_rules.md` | **Yes** — `~/.windsurf/worktrees/<repo_name>` |
| **Cline** | **No** | `.cline/skills/`, `.clinerules/skills/`, `.claude/skills/` | `~/.cline/skills/` | `.clinerules/` (`*.md`/`*.txt`), `AGENTS.md`; user `~/Documents/Cline/Rules`, `~/.agents/AGENTS.md` | **Yes** (Cline Kanban) |
| **Roo Code** | **Yes** | `.roo/skills/`, `.agents/skills/` (+ `-{mode}` variants) | `~/.roo/skills/`, `~/.agents/skills/` | `.roo/rules/`, `.roorules`, `AGENTS.md`; user `~/.roo/rules/` | **Yes** (+ `.worktreeinclude`). *Extension shut down 2026-05-15* |
| **Kilo Code** | **Yes** (project) | `.kilo/skills/`, `.agents/skills/`, `.claude/skills/` | `~/.kilo/skills/` | `AGENTS.md`/`AGENT.md`, `.kilo/rules/*.md` via `kilo.jsonc` `instructions`; user `~/.config/kilo/kilo.jsonc` | No user-facing feature documented |
| **Zed** | **Yes — the only skills root** | `<worktree>/.agents/skills/` | `~/.agents/skills/` | `AGENTS.md`; user `~/.config/zed/AGENTS.md` (`%APPDATA%\Zed\AGENTS.md`) | **Yes** (worktree picker; task hook `create_worktree`) |
| **Amazon Q Developer** | No (skills not documented) | `.amazonq/rules/*.md` | Not documented (UNVERIFIED) | `.amazonq/rules/*.md` | Not documented (UNVERIFIED) |
| **Aider** | No (no skills) | — | — | `CONVENTIONS.md` via `--read`; `.aider.conf.yml` `read:` | Not documented (UNVERIFIED) |
| **Continue** | No (no skills) | — | — | `.continue/rules/*.md` | Not documented (UNVERIFIED) |
| **JetBrains Junie** | **Yes** | `.junie/skills/`, `.agents/skills/` (trusted) | `~/.junie/skills/`, `~/.agents/skills/` | `.junie/AGENTS.md`, `AGENTS.md`; user `~/.junie/AGENTS.md` | **Yes** — sibling `../<project>-junie-wt-NN` |
| **Google Antigravity** | **Yes — the default** | `<workspace-root>/.agents/skills/` | `~/.gemini/config/skills/` (2.0) / `~/.gemini/antigravity/skills/` (IDE) **DISCREPANCY** | `.agents/rules/` (`.agent/rules` legacy); user `~/.gemini/GEMINI.md` | **Yes** ("New Worktree Mode"; path not documented) |
| **pi** (`badlogic/pi-mono`) | **Yes** | `.agents/skills/` (cwd + ancestors), `.pi/skills/` | `~/.agents/skills/`, `~/.pi/agent/skills/` | `AGENTS.md`/`CLAUDE.md`, `AGENTS.override.md`; user `~/.pi/agent/AGENTS.md` | No — extension territory only |

### 1b. Summary table — hooks

| Harness | Hook config (project) | Hook config (user/global) | Format | Events | Payload | Timeout / blocking |
| --- | --- | --- | --- | --- | --- | --- |
| **Claude Code** | `.claude/settings.json`, `.claude/settings.local.json` | `~/.claude/settings.json`; managed policy JSON | JSON, `hooks` → event → matcher groups → handlers | **33** | stdin JSON | default **600 s** (30 s on `UserPromptSubmit`, 10 s `MessageDisplay`); `SessionEnd` 1.5 s budget (≤60 s); **agent waits**; exit 2 blocks; `async: true` escapes |
| **Codex CLI** | `<repo>/.codex/hooks.json` or `[hooks]` in `.codex/config.toml` | `~/.codex/hooks.json` / `~/.codex/config.toml` | JSON **or TOML** | **12** | stdin JSON (one object) | default **600 s**; `SessionEnd`/`Interrupt` 1 s, max 3 s; synchronous by default; hooks require **trust** (hash) |
| **Cursor** | `<project>/.cursor/hooks.json` | `~/.cursor/hooks.json`; enterprise MDM JSON | JSON (`version`, `hooks`) | **21** (18 agent, 2 Tab, 1 app) | stdin JSON, per-event fields | `timeout` seconds, "platform default" (no numeric default documented — **UNVERIFIED**); **fail-open by default** (`failClosed: true` to block); exit 2 blocks |
| **GitHub Copilot (VS Code)** | `.github/hooks/*.json`; Claude format in `.claude/settings.json` | `~/.copilot/hooks`, `~/.claude/settings.json` | JSON, Claude Code-shaped | **8** | stdin JSON | default **30 s**; blocking supported; workspace overrides user per event |
| **Gemini CLI** | `.gemini/settings.json` | `~/.gemini/settings.json`; `/etc/gemini-cli/settings.json` | JSON, `hooks` → event → matcher group → handlers | **11** | stdin JSON (snake_case) | `timeout` **milliseconds**, default **60000**; **runs synchronously, blocks the loop**; exit 2 blocks; `SessionEnd` not awaited |
| **opencode** | `.opencode/plugins/*.ts` | `~/.config/opencode/plugins/` | TypeScript/JS modules | ~30 bus events + `tool.execute.before/after` | in-process objects | block by `throw`; no timeout documented |
| **Windsurf / Devin Desktop** | `.windsurf/hooks.json` | `~/.codeium/windsurf/hooks.json`; system `…/Windsurf/hooks.json` | JSON (`hooks` → event → array) | **12** | stdin JSON | no timeout documented (**UNVERIFIED**); exit 2 blocks pre-hooks only |
| **Cline** | `.cline/plugins/` | `~/.cline/plugins/` | TS/JS plugin modules | 7 hooks / 15 stages | objects | `mode: "blocking"`, `failureMode` |
| **Kilo Code** | `.kilo/plugin/` (legacy `.kilocode/plugin/`) | `~/.config/kilo/plugin/` | TS/JS plugin modules | ~19 | objects | block by `throw`; mirrors opencode |
| **Zed** | `tasks.json` (hook field on a task template) | same file, global scope | JSON | 1 (`create_worktree`) | **env vars**, not stdin | no timeout documented |
| **Amazon Q Developer** | agent JSON `hooks` field | (agent config) | JSON | **5** | stdin (payload schema undocumented) | `preToolUse` can block; no timeout documented |
| **JetBrains Junie** | — | — | — | none found | — | — |
| **Google Antigravity** | `.agents/hooks.json` | `~/.gemini/config/hooks.json` | JSON, name-keyed | **5** | stdin JSON (camelCase) | `timeout` **seconds**, default **30** |
| **pi** | `.pi/extensions/*.ts` | `~/.pi/agent/extensions/*.ts` | TypeScript extensions | ~30 | in-process objects | `return { block: true }`; no timeout documented |
| **Roo Code / Aider / Continue** | none found | none found | — | — | — | — |

---

## 2. The `.agents/skills` convention — who actually reads it

This is berth's current backbone (`internal/skill/skill.go` installs only to
`<repo>/.agents/skills/berth/SKILL.md`). berth's README claims that location is "read by Cursor, Codex, pi
and Antigravity". **All four claims are correct**, and the convention is wider than four harnesses.

### Confirmed readers (first-party documentation)

| Harness | Literal documented line | Source |
| --- | --- | --- |
| **Codex CLI** | Table: `REPO` = `$CWD/.agents/skills`, `$CWD/../.agents/skills`, `$REPO_ROOT/.agents/skills`; `USER` = `$HOME/.agents/skills`. "Codex scans `.agents/skills` in every directory from your current working directory up to the repository root." | https://learn.chatgpt.com/codex/skills (linked from https://raw.githubusercontent.com/openai/codex/main/docs/skills.md) |
| **Cursor** | "Skills are automatically loaded from these locations: `.agents/skills/` (Project-level), `.cursor/skills/` (Project-level), `~/.agents/skills/` (User-level), `~/.cursor/skills/` (User-level)." Also: "Cursor also loads skills from Claude and Codex directories: `.claude/skills/`, `.codex/skills/`, `~/.claude/skills/`, and `~/.codex/skills/`." | https://cursor.com/docs/skills |
| **pi** | "Pi loads skills from: Global: `~/.pi/agent/skills/`, `~/.agents/skills/` — Project (only after the project is trusted): `.pi/skills/`, `.agents/skills/` in `cwd` and ancestor directories" | https://raw.githubusercontent.com/badlogic/pi-mono/main/packages/coding-agent/docs/skills.md |
| **Antigravity** | "`<workspace-root>/.agents/skills/<skill-folder>/` — Workspace-specific"; "Antigravity now defaults to .agents/skills, but still maintains backward support for .agent/skills." | https://antigravity.google/docs/skills |
| **Gemini CLI** | "User skills: Located in `~/.gemini/skills/` or the `~/.agents/skills/` alias. Workspace skills: Located in `.gemini/skills/` or the `.agents/skills/` alias." and "Within the same tier (user or workspace), the `.agents/skills/` alias takes precedence over the `.gemini/skills/` directory." | https://geminicli.com/docs/cli/skills/ |
| **GitHub Copilot** | "Project skills, stored in your repository (`.github/skills`, `.claude/skills`, or `.agents/skills`); Personal skills … (`~/.copilot/skills` or `~/.agents/skills`)" | https://docs.github.com/en/copilot/concepts/agents/about-agent-skills |
| **opencode** | "Project agent-compatible: `.agents/skills/<name>/SKILL.md`; Global agent-compatible: `~/.agents/skills/<name>/SKILL.md`" | https://raw.githubusercontent.com/sst/opencode/dev/packages/web/src/content/docs/skills.mdx |
| **Windsurf / Devin Desktop** | "For cross-agent compatibility, Devin Desktop also discovers skills in `.agents/skills/` and `~/.agents/skills/`." | https://docs.windsurf.com/windsurf/cascade/skills |
| **Roo Code** | "Project skills … `<project-root>/.agents/skills/{skill-name}/SKILL.md` … Global skills … `~/.agents/skills/{skill-name}/SKILL.md`" | https://roocodeinc.github.io/Roo-Code/features/skills/ |
| **Kilo Code** | "For interoperability with other tools, Kilo Code also loads skills from: `.agents/skills/` — Open agent standard, loaded by default" | https://kilo.ai/docs/customize/skills |
| **Zed** | "Global `~/.agents/skills/` … Project-local `<worktree>/.agents/skills/`"; "Skills are loaded from `~/.agents/skills/` and `<worktree>/.agents/skills/` only." | https://zed.dev/docs/ai/skills |
| **JetBrains Junie** | "Junie CLI also loads skills from: `.agents/skills/` directories: `<projectRoot>/.agents/skills/` (in a trusted project) and `~/.agents/skills/`" | https://junie.jetbrains.com/docs/agent-skills.html |

That is **12 harnesses**, not 4 — including Zed, where `.agents/skills` is the *only* skills root.

### Documented non-readers

| Harness | Evidence | Source |
| --- | --- | --- |
| **Claude Code** | Its skills location table (Enterprise / Personal / Project / Nested / `--add-dir` / Plugin / claude.ai) does not include `.agents/skills`. A grep of the fetched skills, memory, plugins, plugins-reference, claude-directory, large-codebases, features-overview, agent-sdk/skills, settings, hooks, managed-settings, sub-agents, agent-teams, worktrees and llms.txt pages for `.agents` / `agents/skills` found **zero** occurrences. Its instruction file is `CLAUDE.md`: "Claude Code reads `CLAUDE.md`, not `AGENTS.md`." | https://code.claude.com/docs/en/skills.md , https://code.claude.com/docs/en/memory.md |
| **Cline** | "Project skills: `.cline/skills/` (recommended), `.clinerules/skills/`, `.claude/skills/`. Global skills: `~/.cline/skills/`". `.agents/skills` is absent. **DISCREPANCY**: the `skills` CLI lists Cline's project path as `.agents/skills/` (secondary source). | https://docs.cline.bot/customization/skills.md vs https://raw.githubusercontent.com/vercel-labs/skills/main/README.md |
| **Aider** | No skills page; documentation covers `CONVENTIONS.md` and `.aider.conf.yml` only. | https://aider.chat/docs/usage/conventions.html |
| **Continue** | No Skills page in the docs navigation; Customize contains only Models, MCP servers, Rules, Prompts. | https://docs.continue.dev/customize/rules |
| **Amazon Q Developer** | No first-party page documents agent skills for Q or the Q CLI. | https://aws.github.io/amazon-q-developer-cli/agent-format.html |

Caveat that applies to every "No" in this section: **docs rarely state a documented negative.** "Claude
Code ignores `.agents/skills`" is *not proven* — it is simply absent from a table that presents itself as
the complete set. Treat these as "do not rely on it", not "impossible".

### Zoneless formats — the conventions that are *not* `.agents/skills`

- **`AGENTS.md`** is the near-universal *instruction* contract: opencode, Cline, Roo, Kilo, pi,
  Windsurf/Devin Desktop, Zed, Junie, Copilot (VS Code), Antigravity (rules engine, unverified for
  Antigravity). Claude Code is the notable hold-out and needs `CLAUDE.md`.
- **`.claude/skills`** is the other de-facto compatibility root: Cursor, opencode, Copilot, Cline, Kilo
  read it; Junie detects and offers to import it.

---

## 3. Per-harness detail

Format: discovery → instructions → hooks (with a literal merge snippet where one applies) → worktrees.

### 3.1 Claude Code

- **Skills.** Project `.claude/skills/<skill-name>/SKILL.md`; personal `~/.claude/skills/<skill-name>/SKILL.md`;
  nested `<subdir>/.claude/skills/…`; plugin `<plugin>/skills/<skill-name>/SKILL.md`; enterprise in the
  managed settings directory. Discovery walks `.claude/skills/` "in the directory where you start it and in
  every parent directory up to the repository root". `SKILL.md` is required; the skill folder name becomes
  the command. `[source: https://code.claude.com/docs/en/skills.md]`
- **Frontmatter.** "All fields are optional. Only `description` is recommended". Documented keys: `name`,
  `description`, `when_to_use`, `argument-hint`, `arguments`, `disable-model-invocation`, `user-invocable`,
  `allowed-tools`, `disallowed-tools`, `model`, `effort`, `context`, `agent`, `background`, `hooks`, `paths`,
  `shell`, `metadata`, `license`, `compatibility`. Frontmatter is read "only when the opening `---` is the
  file's first line". Only `name`, `description`, `license`, `compatibility`, `metadata`, `allowed-tools`
  survive claude.ai upload / Skills API / `package_skill.py`; a stray key is a hard error.
  `[source: https://code.claude.com/docs/en/skills.md]`
- **Instructions.** Managed policy `C:\Program Files\ClaudeCode\CLAUDE.md` (Windows) /
  `/etc/claude-code/CLAUDE.md` (Linux) / `/Library/Application Support/ClaudeCode/CLAUDE.md` (macOS); user
  `~/.claude/CLAUDE.md`; project `./CLAUDE.md` or `./.claude/CLAUDE.md`; local `./CLAUDE.local.md`. All are
  concatenated, not overridden; parent directories are walked. `.claude/rules/*.md` is also loaded
  ("Rules without `paths` frontmatter are loaded at launch with the same priority as `.claude/CLAUDE.md`");
  user rules live in `~/.claude/rules/`. `[source: https://code.claude.com/docs/en/memory.md]`
- **Hooks.** Config: `~/.claude/settings.json`, `.claude/settings.json`, `.claude/settings.local.json`,
  managed policy settings, plugin `hooks/hooks.json`, skill/subagent frontmatter.
  **33 events** (identical list on the hooks page and the plugins reference): `SessionStart`, `Setup`,
  `UserPromptSubmit`, `UserPromptExpansion`, `PreToolUse`, `PermissionRequest`, `PermissionDenied`,
  `PostToolUse`, `PostToolUseFailure`, `PostToolBatch`, `Notification`, `MessageDisplay`, `SubagentStart`,
  `SubagentStop`, `TaskCreated`, `TaskCompleted`, `Stop`, `StopFailure`, `TeammateIdle`,
  `InstructionsLoaded`, `ConfigChange`, `CwdChanged`, `DirectoryAdded`, `FileChanged`, `WorktreeCreate`,
  `WorktreeRemove`, `PreCompact`, `PostCompact`, `PreModelSwitch`, `PostModelSwitch`, `Elicitation`,
  `ElicitationResult`, `SessionEnd`. Handlers: `command`, `http`, `mcp_tool`, `prompt`, `agent`.
  Payload: stdin JSON with `session_id`, `prompt_id`, `transcript_path`, `cwd`, `scratchpad_dir`,
  `permission_mode`, `effort`, `hook_event_name`. `[source: https://code.claude.com/docs/en/hooks.md]`
  A worktree-oriented entry, as a tool would merge it (the docs' own merge instruction is "add
  `Notification` as a sibling of the existing event keys rather than replacing the whole object"):

  ```json
  {
    "hooks": {
      "WorktreeCreate": [
        { "hooks": [ { "type": "command", "command": "\"$CLAUDE_PROJECT_DIR\"/.claude/hooks/worktree.sh" } ] }
      ]
    }
  }
  ```

  `[source: https://code.claude.com/docs/en/hooks-guide.md]`
- **Blocking/timeout.** "All matching hooks run in parallel"; the agent waits for them. `timeout` in
  **seconds**, default **600** for `command`. "Claude Code lowers the `command` … default to 30 on
  `UserPromptSubmit`, `PreModelSwitch`, and `PostModelSwitch`, and to 10 on `MessageDisplay`. `SessionEnd`
  hooks share a 1.5-second budget; if your settings set a longer per-hook `timeout`, Claude Code raises the
  budget to match, up to 60 seconds." Exit `2` blocks; exit `1` does **not**. `async: true` "runs in the
  background without blocking" and is exempt from the timeout. `WorktreeCreate` is special: "any non-zero
  exit code from `WorktreeCreate` aborts worktree creation", and a `WorktreeCreate` hook can "replace the
  default `git worktree` logic entirely". `[source: https://code.claude.com/docs/en/hooks.md]`
- **Mergeability.** "Hook entries merge across settings levels rather than replacing each other"; "Hooks
  merge: all registered hooks fire for their matching events regardless of source". List keys combine across
  files. **There is no documented API, read-modify-write contract, atomicity or unknown-key preservation
  guarantee for third-party tools** — the docs only say to "edit the settings JSON directly".
  `[source: https://code.claude.com/docs/en/hooks.md]` `[source: https://code.claude.com/docs/en/settings.md]`
- **Worktrees.** `claude --worktree <name>` (`-w`) creates `.claude/worktrees/<name>/` on branch
  `worktree-<name>`; `--worktree "#1234"` creates `.claude/worktrees/pr-<number>`; subagents with
  `isolation: worktree`; the desktop app gives every session a worktree; a `WorktreeCreate` hook fires "When
  a worktree is being created via `--worktree`, `isolation: "worktree"`, or for a background session".
  `${CLAUDE_PROJECT_DIR}` deliberately does *not* follow the worktree, but the hook input's `cwd` does.
  `[source: https://code.claude.com/docs/en/worktrees.md]`

### 3.2 OpenAI Codex CLI

- **Skills.** Repo: `.agents/skills` scanned "in every directory from your current working directory up to
  the repository root" (documented rows `$CWD/.agents/skills`, `$CWD/../.agents/skills`,
  `$REPO_ROOT/.agents/skills`). User `$HOME/.agents/skills`. Admin `/etc/codex/skills`. System skills are
  bundled and cached at `$CODEX_HOME/skills/.system`. **There is no documented `~/.codex/skills` authoring
  location.** Required frontmatter: `name`, `description` (exactly those two). Symlinked skill folders are
  followed. `[source: https://learn.chatgpt.com/codex/skills]`
  Source corroboration for the system cache only: `const SKILLS_DIR_NAME: &str = "skills";`,
  `const SYSTEM_SKILLS_DIR_NAME: &str = ".system";` `[source: https://raw.githubusercontent.com/openai/codex/main/codex-rs/skills/src/lib.rs]`
- **Instructions.** `AGENTS.override.md` before `AGENTS.md`, then `project_doc_fallback_filenames`; at most
  one file per directory; concatenated root → cwd; `project_doc_max_bytes` default 32 KiB. Global: `~/.codex/AGENTS.override.md` else `~/.codex/AGENTS.md`. Config is TOML at
  `~/.codex/config.toml` and `.codex/config.toml`. `[source: https://learn.chatgpt.com/docs/agent-configuration/agents-md]`
- **Hooks.** Locations: `~/.codex/hooks.json`, `~/.codex/config.toml`, `<repo>/.codex/hooks.json`,
  `<repo>/.codex/config.toml`. **12 events**: `PreToolUse`, `PermissionRequest`, `PostToolUse`,
  `PreCompact`, `PostCompact`, `UserPromptSubmit`, `SubagentStop`, `Stop`, `Interrupt`, `SessionStart`,
  `SubagentStart`, `SessionEnd`. "Every command hook receives one JSON object on `stdin`" with `session_id`,
  `transcript_path`, `cwd`, `hook_event_name`, `model`, and `permission_mode` on most events.
  `[source: https://learn.chatgpt.com/docs/hooks]`

  ```toml
  [[hooks.PreToolUse]]
  matcher = "^Bash$"

  [[hooks.PreToolUse.hooks]]
  type = "command"
  command = '/usr/bin/python3 "$(git rev-parse --show-toplevel)/.codex/hooks/pre_tool_use_policy.py"'
  timeout = 30
  statusMessage = "Checking Bash command"
  ```

  `[source: https://learn.chatgpt.com/docs/config-file/config-advanced]`
- **Blocking/timeout.** "If `timeout` is omitted, Codex uses `600` seconds for most hooks." "`SessionEnd`
  and `Interrupt` use `1` second by default and support up to `3` seconds." "By default, Codex waits for a
  command hook to finish before continuing the operation that triggered it." Matching hooks launch
  concurrently. Hooks are **trust-gated**: "Codex records trust against the hook's current hash, so new or
  changed hooks are marked for review and skipped until trusted." `[source: https://learn.chatgpt.com/docs/hooks]`
- **Mergeability.** "Matching hooks from multiple files all run." / "Higher-precedence config layers don't
  replace lower-precedence hooks." Project-local hooks load only when the `.codex/` layer is trusted.
  `[source: https://learn.chatgpt.com/docs/hooks]`
- **Worktrees.** **UNVERIFIED** — no first-party Codex page mentions `git worktree`; cloud tasks are
  documented as containers. `[source: https://learn.chatgpt.com/docs/cloud.md]`

### 3.3 Cursor

- **Skills.** Project: `.agents/skills/`, `.cursor/skills/`. User: `~/.agents/skills/`, `~/.cursor/skills/`.
  Compatibility reads: `.claude/skills/`, `.codex/skills/`, `~/.claude/skills/`, `~/.codex/skills/`.
  Recursive: "Cursor walks the skills root recursively and picks up any `SKILL.md` it finds", and a
  `.cursor/skills/` or `.agents/skills/` folder anywhere in the repo is picked up (monorepos).
  `SKILL.md` + optional `scripts/`, `references/`, `assets/`. Frontmatter: `name` (required, "Lowercase
  letters, numbers, and hyphens only. **Must match the parent folder name.**"), `description` (required),
  and optional `paths`, `disable-model-invocation`, `icon`, `color`, `metadata`; "The legacy `globs` field is
  still accepted as a fallback for older skills, but new skills should use `paths`."
  `[source: https://cursor.com/docs/skills]`
- **Instructions.** `.cursor/rules/*.mdc` — "Project rules must use the `.mdc` extension. A plain `.md` file
  in `.cursor/rules` is ignored". Frontmatter `description`, `globs`, `alwaysApply` with four behaviours,
  named in the docs as `Always Apply` / `Apply Intelligently` / `Apply to Specific Files` / `Apply
  Manually`. Precedence: "Team Rules → Project Rules → User Rules. All applicable rules are merged; earlier
  sources take precedence when guidance conflicts." `AGENTS.md` is supported ("Cursor supports AGENTS.md in
  the project root and subdirectories… Instructions from nested `AGENTS.md` files are combined with parent
  directories, with more specific instructions taking precedence"), and so is `CLAUDE.md` ("Cursor reads
  `CLAUDE.md` files the same way it reads `AGENTS.md`… always applied to every conversation, regardless of
  any `alwaysApply` frontmatter setting"). Legacy `.cursorrules` at the project root is deprecated.
  `[source: https://cursor.com/docs/rules]` `[source: https://cursor.com/help/customization/rules]`
- **User rules — two distinct things.** Rules stored in the account via **Customize → Rules** sync across
  machines; rule *files* in `~/.cursor/rules` (`%USERPROFILE%\.cursor\rules` on Windows) "stay on the
  machine and do not sync".
  `[source: https://cursor.com/help/customization/rules]` `[source: https://cursor.com/changelog/2-1]`
- **Which surfaces create worktrees.** Agents Window (UI-native feature, worktree picker); IDE via the
  `/worktree`, `/best-of-n`, `/apply-worktree`, `/delete-worktree` slash commands; the CLI via
  `-w` / `--worktree [name]` (`--skip-worktree-setup` skips the setup file, `--worktree-base <branch>`
  changes the base). Worktrees land in `~/.cursor/worktrees/<reponame>/<name>`. Subagents each get "an
  isolated Git worktree with a separate working directory on the same machine". **Cloud agents do not use
  worktrees** — they "clone your repo … and work on a separate branch" inside a dedicated VM. The CLI binary
  is documented as `agent`, not `cursor-agent`.
  `[source: https://cursor.com/docs/cli/using]` `[source: https://cursor.com/docs/subagents]`
  `[source: https://cursor.com/docs/cloud-agent]`
  Ordering caveat: the setup file is documented as executed *at* worktree creation ("executed sequentially in
  the worktree") and debug output has a `Worktrees Setup` channel, but no page states literally that setup
  completes **before** the agent starts — **UNVERIFIED as a literal guarantee**.
- **Hooks.** Project `<project>/.cursor/hooks.json`; user `~/.cursor/hooks.json`; enterprise MDM
  `/Library/Application Support/Cursor/hooks.json` (macOS), `/etc/cursor/hooks.json` (Linux/WSL),
  `C:\ProgramData\Cursor\hooks.json` (Windows). **20 events** — agent: `sessionStart`, `sessionEnd`,
  `preToolUse`, `postToolUse`, `postToolUseFailure`, `subagentStart`, `subagentStop`,
  `beforeShellExecution`, `afterShellExecution`, `beforeMCPExecution`, `afterMCPExecution`, `beforeReadFile`,
  `afterFileEdit`, `beforeSubmitPrompt`, `preCompact`, `stop`, `afterAgentResponse`, `afterAgentThought`;
  Tab: `beforeTabFileRead`, `afterTabFileEdit`; app: `workspaceOpen`. Payload: JSON on **stdin**, per-event
  fields (`beforeShellExecution` gets `{command, cwd, sandbox}`). Exit `2` blocks; other exit codes proceed.
  `[source: https://cursor.com/docs/hooks]`

  ```json
  { "version": 1, "hooks": { "afterFileEdit": [{ "command": ".cursor/hooks/format.sh" }] } }
  ```

  Global options: `version` (default `1`). Per-script: `command`, `type` (`"command"`|`"prompt"`),
  `timeout` (seconds, "platform default"), `loop_limit` (default 5), `failClosed` (default `false`),
  `matcher`. `[source: https://cursor.com/docs/hooks]`
- **Blocking/timeout.** **Fail-open by default**: "hook failures (crash, timeout, invalid JSON) allow the
  action through". `failClosed: true` inverts that. `timeout` is "Execution timeout in seconds" with a
  "platform default" value that the page never states — **UNVERIFIED**.
  `[source: https://cursor.com/docs/hooks]`
- **Mergeability.** "All matching hooks from every source run; when responses conflict, higher-priority
  sources take precedence during merge". Priority: enterprise → team → project `.cursor/hooks.json` → user
  `~/.cursor/hooks.json` → Claude `.claude/settings.local.json` → `.claude/settings.json` → `~/.claude/settings.json`.
  Cursor **also loads Claude Code hooks** from the `.claude/` files, gated by "Include third-party Plugins,
  Skills, and other configs" in Settings.
  `[source: https://cursor.com/docs/reference/third-party-hooks]`
- **Worktrees.** Cursor creates worktrees in the **Agents Window**, the **IDE**, and the **CLI**, and "Cursor
  checks this file when it creates a worktree in the Agents Window, the IDE, or the Cursor CLI". Setup file
  `.cursor/worktrees.json`, looked up first "in the worktree path", then "in the root path of your project".
  Keys: `setup-worktree-unix`, `setup-worktree-windows`, `setup-worktree` (generic fallback); each accepts
  an array of shell commands executed sequentially, or a string script path relative to the JSON file.
  `$ROOT_WORKTREE_PATH` points at the original checkout. `[source: https://cursor.com/docs/configuration/worktrees]`
  berth's current adapter matches this schema exactly (`internal/skill/adapters/cursor.worktrees.json`).
  Ambiguity worth quoting: "The UI-native worktrees feature described on this page is only available in the
  Agents Window. In the IDE, use the Worktree Skills commands below." The page never defines "Worktree
  Skills commands"; the same page simultaneously says the file is read by the IDE and CLI.

### 3.4 GitHub Copilot (VS Code agent mode + cloud agent)

- **Skills.** "Skill files must be named `SKILL.md`." Project: `.github/skills/`, `.claude/skills/`,
  `.agents/skills/`. Personal: `~/.copilot/skills/`, `~/.claude/skills/`, `~/.agents/skills/`. Frontmatter
  `name` (required, "Must match the parent directory name. Maximum 64 characters. Names with invalid
  characters cause the skill to silently fail to load"), `description` (required, max 1024), `license`,
  `allowed-tools`, plus VS Code extras `argument-hint`, `user-invocable`, `disable-model-invocation`,
  `context` (experimental). Works with "Copilot cloud agent, Copilot code review, the GitHub Copilot CLI, the
  GitHub Copilot app, and agent mode in Visual Studio Code and JetBrains IDEs".
  `[source: https://docs.github.com/en/copilot/how-tos/copilot-on-github/customize-copilot/customize-cloud-agent/add-skills]`
  `[source: https://code.visualstudio.com/docs/agent-customization/agent-skills]`
- **Instructions.** `.github/copilot-instructions.md` (repo-wide);
  `.github/instructions/NAME.instructions.md` with `applyTo` frontmatter; `.claude/rules` (uses `paths`
  instead of `applyTo`); `AGENTS.md` (`chat.useAgentsMdFile`, nested is experimental); `CLAUDE.md`,
  `.claude/CLAUDE.md`, `CLAUDE.local.md`. User: `~/.copilot/instructions` or `~/.claude/rules`. Priority:
  "1. Personal instructions (user-level, highest priority) 2. Repository instructions
  (`.github/copilot-instructions.md` or `AGENTS.md`) 3. Organization instructions (lowest priority)", and
  "If you have multiple instruction files in your project, VS Code combines and adds them to the chat
  context, no specific order is guaranteed". `github.copilot.chat.codeGeneration.instructions` is deprecated
  as of VS Code 1.102. `[source: https://code.visualstudio.com/docs/agent-customization/custom-instructions]`
- **Hooks.** VS Code: workspace `.github/hooks/*.json`, Claude format in `.claude/settings.json` /
  `.claude/settings.local.json`; user `~/.copilot/hooks`, `~/.claude/settings.json`; custom-agent `hooks`
  frontmatter; plugins `hooks.json`. **8 events**: `SessionStart`, `UserPromptSubmit`, `PreToolUse`,
  `PostToolUse`, `PreCompact`, `SubagentStart`, `SubagentStop`, `Stop` — note there is **no `SessionEnd`**.
  "VS Code uses the same hook format as Claude Code and Copilot CLI for compatibility":
  `{ "hooks": { "PreToolUse": [ { "type": "command", "command": "./scripts/validate-tool.sh", "timeout": 15 } ] } }`
  Properties: `type`, `command`, `windows`, `linux`, `osx`, `cwd`, `env`, `timeout` ("Timeout in seconds
  (default: 30)"). `[source: https://code.visualstudio.com/docs/agent-customization/hooks]`
  `[source: https://code.visualstudio.com/docs/agents/reference/hooks-reference]`
- **Blocking/timeout.** Default 30 s. Exit `2` = "Blocking error"; other non-zero = warning. "If multiple
  matching hooks run for the same tool invocation, the most restrictive decision wins". "Currently, VS Code
  ignores matcher values, so hooks run on all tool invocations regardless of the matcher."
  `[source: https://code.visualstudio.com/docs/agent-customization/hooks]`
- **Mergeability.** "Workspace hooks take precedence over user hooks for the same event type." Default
  `chat.hookFilesLocations` includes `.github/hooks`, `.claude/settings.local.json`, `.claude/settings.json`,
  `~/.claude/settings.json`. `[source: https://code.visualstudio.com/docs/agent-customization/hooks]`
- **Worktrees.** **UNVERIFIED** for both VS Code agent mode and the cloud agent. The cloud agent's
  environment is "ephemeral development environment, powered by GitHub Actions", configured by
  `.github/workflows/copilot-setup-steps.yml` whose single job must be named `copilot-setup-steps`; if you
  do not check out the repo, "Copilot will do this for you automatically after the steps complete".
  `[source: https://docs.github.com/en/copilot/how-tos/copilot-on-github/customize-copilot/customize-cloud-agent/customize-the-agent-environment]`

### 3.5 Gemini CLI

- **Skills.** Tiers lowest→highest: built-in → extension → user (`~/.gemini/skills/` or `~/.agents/skills/`)
  → workspace (`.gemini/skills/` or `.agents/skills/`). The `.agents/skills/` alias **wins inside a tier**.
  Toggle `skills.enabled` (default `true`, not under `experimental`). Frontmatter: `name` ("should match the
  directory name") and `description`. Activation calls an `activate_skill` tool behind a consent prompt.
  Management: `gemini skills list --all`, `gemini skills install <url> --consent`, `gemini skills link .`,
  `gemini skills uninstall <name> --scope workspace`, and `/skills …` slash commands.
  `[source: https://geminicli.com/docs/cli/skills/]`
  `[source: https://raw.githubusercontent.com/google-gemini/gemini-cli/main/docs/cli/creating-skills.md]`
- **Instructions.** `~/.gemini/GEMINI.md`; `GEMINI.md` in the workspace and parent directories "up to either
  the project root (identified by a `.git` folder) or your home directory"; JIT scanning for
  tool-accessed paths, capped by `context.discoveryMaxDirs` (default 200). The key is **`context.fileName`
  and it is an array**: `{"context": {"fileName": ["AGENTS.md", "CONTEXT.md", "GEMINI.md"]}}`. Settings:
  `~/.gemini/settings.json`, `your-project/.gemini/settings.json`; "Workspace settings override user
  settings". `[source: https://raw.githubusercontent.com/google-gemini/gemini-cli/main/docs/cli/gemini-md.md]`
- **Hooks.** Configured in `settings.json`. Precedence: project `.gemini/settings.json` → user
  `~/.gemini/settings.json` → system `/etc/gemini-cli/settings.json` → extensions. **11 events**:
  `SessionStart`, `SessionEnd`, `BeforeAgent`, `AfterAgent`, `BeforeModel`, `AfterModel`,
  `BeforeToolSelection`, `BeforeTool`, `AfterTool`, `PreCompress`, `Notification`. Schema:
  `{"hooks":{"BeforeTool":[{"matcher":"write_file|replace","hooks":[{"name":"security-check","type":"command","command":"$GEMINI_PROJECT_DIR/.gemini/hooks/security.sh","timeout":5000}]}]}}`.
  Handler fields: `type` ("Currently only `\"command\"` is supported"), `command`, `name`, `timeout`,
  `description`; group fields `matcher`, `sequential`, `hooks`. Base stdin JSON: `session_id`,
  `transcript_path`, `cwd`, `hook_event_name`, `timestamp`. Env vars include `GEMINI_PROJECT_DIR`,
  `GEMINI_SESSION_ID`, `GEMINI_CWD` and `CLAUDE_PROJECT_DIR` "(Alias) Provided for compatibility".
  `[source: https://raw.githubusercontent.com/google-gemini/gemini-cli/main/docs/hooks/index.md]`
  `[source: https://raw.githubusercontent.com/google-gemini/gemini-cli/main/docs/hooks/reference.md]`
- **Blocking/timeout.** "Hooks run synchronously as part of the agent loop—when a hook event fires, Gemini
  CLI waits for all matching hooks to complete before continuing." `timeout` is in **milliseconds**,
  "default: 60000". Exit `0` = parse stdout JSON; `2` = "System Block"; other = warning. "Silence is
  Mandatory: Your script must not print any plain text to `stdout` other than the final JSON" — otherwise
  the CLI "will default to 'Allow'". `SessionEnd` is "**Best Effort**: The CLI **will not wait** for this
  hook to complete". `[source: https://raw.githubusercontent.com/google-gemini/gemini-cli/main/docs/hooks/reference.md]`
- **Mergeability.** Layers merge with project highest; project hooks are fingerprinted — a changed command
  "is treated as a new, untrusted hook".
  `[source: https://raw.githubusercontent.com/google-gemini/gemini-cli/main/docs/hooks/index.md]`
- **Worktrees.** Experimental: `{"experimental": {"worktrees": true}}` plus `gemini --worktree <name>` (`-w`).
  "The value you pass becomes both the directory name (within `.gemini/worktrees/`) and the branch name."
  It "does not automatically delete your worktree or branch" on exit. Note the docs' own warning that the
  worktree is not provisioned: "Remember to initialize your development environment in each new worktree
  according to your project's setup." `[source: https://geminicli.com/docs/cli/git-worktrees/]`

### 3.6 opencode

- **Skills.** Six roots: `.opencode/skills/`, `~/.config/opencode/skills/`, `.claude/skills/`,
  `~/.claude/skills/`, `.agents/skills/`, `~/.agents/skills/`. "For project-local paths, OpenCode walks up
  from your current working directory until it reaches the git worktree." Recognised frontmatter: `name`
  and `description` required, `license`, `compatibility`, `metadata` optional; "Unknown frontmatter fields
  are ignored". Loaded via the `skill` tool; `permission.skill` globs (`allow`/`deny`/`ask`).
  `[source: https://raw.githubusercontent.com/sst/opencode/dev/packages/web/src/content/docs/skills.mdx]`
- **Instructions.** Project `AGENTS.md`, global `~/.config/opencode/AGENTS.md`, `CLAUDE.md` fallback, plus an
  `instructions` array that accepts globs and remote URLs ("Remote instructions are fetched with a 5 second
  timeout"). Precedence: "1. Local files by traversing up … 2. Global file … 3. Claude Code file … The first
  matching file wins in each category." Config is `opencode.json`/`opencode.jsonc` and "Configuration files
  are merged together, not replaced."
  `[source: https://raw.githubusercontent.com/sst/opencode/dev/packages/web/src/content/docs/rules.mdx]`
- **Hooks.** Not JSON — TypeScript/JS plugins in `.opencode/plugins/` and `~/.config/opencode/plugins/`.
  Context object is `{ project, client, $, directory, worktree }`; returning a hooks object registers
  handlers such as `tool.execute.before`, `tool.execute.after`, `shell.env`, `event`, and bus events
  (`session.idle`, `session.created`, `permission.asked`, `file.edited`, …). Blocking is by throwing:
  `"tool.execute.before": async (input, output) => { throw new Error("Do not read .env files") }`.
  No timeout is documented. `[source: https://raw.githubusercontent.com/sst/opencode/dev/packages/web/src/content/docs/plugins.mdx]`
  Note: `chat.message`, `permission.ask`, `chat.params` are **not** in opencode's own plugins doc (they
  appear only in Kilo Code's docs) — treat as **UNVERIFIED** for opencode.
- **Worktrees.** **UNVERIFIED** — no worktree page exists; `worktree` appears only as a plugin-context field
  and in the skill-discovery traversal sentence. `[source: https://opencode.ai/sitemap.xml]`

### 3.7 Windsurf / Devin Desktop (Cascade)

Docs now redirect from `docs.windsurf.com/...` to `docs.devin.ai`; `.devin/` is preferred over `.windsurf/`.

- **Skills.** Workspace `.windsurf/skills/<skill-name>/`; global `~/.codeium/windsurf/skills/<skill-name>/`;
  enterprise `C:\ProgramData\Windsurf\skills\` (Windows), `/etc/windsurf/skills/` (Linux/WSL),
  `/Library/Application Support/Windsurf/skills/` (macOS). **"For cross-agent compatibility, Devin Desktop
  also discovers skills in `.agents/skills/` and `~/.agents/skills/`. If you have enabled Claude Code config
  reading, `.claude/skills/` and `~/.claude/skills/` are scanned as well."** Required frontmatter: `name`,
  `description`. `[source: https://docs.windsurf.com/windsurf/cascade/skills]`
- **Instructions.** Global `~/.codeium/windsurf/memories/global_rules.md` ("Always on. Limited to 6,000
  characters"); workspace `.devin/rules/*.md` preferred, `.windsurf/rules/*.md` fallback, legacy
  `.windsurfrules` still read (12,000 chars per rule file); `AGENTS.md` anywhere ("root-level = always-on,
  subdirectory = auto-glob for that directory"); system `/etc/devin/rules/` (legacy `/etc/windsurf/rules/`).
  Frontmatter key is `trigger` with values `always_on`, `model_decision`, `glob` (+ `globs`), `manual`.
  `[source: https://docs.windsurf.com/windsurf/cascade/memories]`
- **Hooks.** Workspace `.windsurf/hooks.json`; user `~/.codeium/windsurf/hooks.json` (Devin Desktop IDE) or
  `~/.codeium/hooks.json` (JetBrains plugin); system `C:\ProgramData\Windsurf\hooks.json`,
  `/etc/windsurf/hooks.json`, `/Library/Application Support/Windsurf/hooks.json`. **12 events**:
  `pre_read_code`, `post_read_code`, `pre_write_code`, `post_write_code`, `pre_run_command`,
  `post_run_command`, `pre_mcp_tool_use`, `post_mcp_tool_use`, `pre_user_prompt`, `post_cascade_response`,
  `post_cascade_response_with_transcript`, **`post_setup_worktree`**. `[source: https://docs.windsurf.com/windsurf/cascade/hooks]`

  ```json
  {
    "hooks": {
      "post_setup_worktree": [
        { "command": "bash $ROOT_WORKSPACE_PATH/hooks/setup_worktree.sh", "show_output": true }
      ]
    }
  }
  ```

  Common stdin fields: `agent_action_name`, `trajectory_id`, `execution_id`, `timestamp`, `model_name`,
  `tool_info`. Per-hook params: `command` (macOS/Linux, `bash -c`), `powershell` (Windows), `show_output`,
  `working_directory`. The documented `post_setup_worktree` payload is:

  ```json
  { "agent_action_name": "post_setup_worktree",
    "tool_info": { "worktree_path": "/Users/me/.windsurf/worktrees/my-repo/abmy-repo-c123",
                   "root_workspace_path": "/Users/me/projects/my-repo" } }
  ```

  `[source: https://docs.windsurf.com/windsurf/cascade/worktrees]`
  Note also: "The hooks described here are Cascade hooks. The Devin Local agent has its own lifecycle hooks
  with a different configuration format." `[source: https://docs.windsurf.com/windsurf/cascade/hooks]`
- **Blocking/timeout.** "For pre-hooks … your script can block the action by exiting with exit code 2";
  "Only pre-hooks (`pre_user_prompt`, `pre_read_code`, `pre_write_code`, `pre_run_command`,
  `pre_mcp_tool_use`) can block actions using exit code 2. Post-hooks cannot block". **No timeout field is
  documented** — **UNVERIFIED**. "Hooks do not load or run while a workspace is open in Restricted Mode."
- **Mergeability.** "Hooks from all three locations are **merged together**. If the same hook event is
  configured in multiple locations, all hooks will execute in order: system → user → workspace."
  `[source: https://docs.windsurf.com/windsurf/cascade/hooks]`
- **Worktrees.** "Worktrees are organized by repo name inside `~/.windsurf/worktrees/<repo_name>`. Each
  worktree is given a unique random name." `post_setup_worktree` "runs after each worktree is created and
  configured. It is executed inside the new **worktree** directory. The `$ROOT_WORKSPACE_PATH` environment
  variable points to the original workspace path". Cleanup: "Each workspace can have up to **20**
  worktrees." `[source: https://docs.windsurf.com/windsurf/cascade/worktrees]`

### 3.8 Cline

- **Skills.** `.cline/skills/` (recommended), `.clinerules/skills/`, `.claude/skills/`; global
  `~/.cline/skills/` (`C:\Users\USERNAME\.cline\skills\` on Windows). Activation via a `use_skill` tool.
  "`name` must exactly match the directory name"; description max 1024 characters. **`.agents/skills` is not
  listed.** Precedence quirk: "When a global skill and project skill have the same name, the global skill
  takes precedence." `[source: https://docs.cline.bot/customization/skills.md]`
- **Instructions.** `.clinerules/` — "Cline processes all `.md` and `.txt` files inside `.clinerules/`";
  `AGENTS.md` and `~/.agents/AGENTS.md` also read; global rules `C:\Users\<you>\Documents\Cline\Rules`
  (Windows) or `~/Documents/Cline/Rules`; "Workspace rules take precedence when they conflict with global
  rules". Conditional rules use `paths:` frontmatter; "Rules without frontmatter are always active."
  `[source: https://docs.cline.bot/customization/cline-rules.md]`
- **Hooks.** The hooks page is a stub pointing at SDK plugins. Current surface: "`beforeRun`, `afterRun`,
  `beforeModel`, `afterModel`, `beforeTool`, `afterTool`, and `onEvent`", with stages
  `session_start`, `run_start`, `iteration_start`, `turn_start`, `before_agent_start`, `tool_call_before`,
  `tool_call_after`, `turn_end`, `stop_error`, `iteration_end`, `run_end`, `session_shutdown`, `error`.
  Plugins load from `~/.cline/plugins/` and `.cline/plugins/`; policy knobs `mode: "blocking"|"async"` and
  `failureMode: "fail_open"|"fail_closed"`. **Caveat:** "This feature currently only applies to Cline SDK,
  CLI, and Kanban. This feature is not applicable on VSCode and JetBrains Extension for now."
  `[source: https://docs.cline.bot/sdk/plugins.md]` `[source: https://docs.cline.bot/customization/plugins.md]`
  The older `TaskStart`/`PreToolUse`/`~/Documents/Cline/Hooks/` hook set is **UNVERIFIED** — absent from the
  current docs.
- **Worktrees.** Yes, via Cline Kanban: "Use Cline Kanban to run multiple coding agents in parallel with
  isolated git worktrees." Checkpoints are a different mechanism (a shadow Git repo).
  `[source: https://docs.cline.bot/llms.txt]`

### 3.9 Roo Code

**Status: "The Roo Code Extension was shut down on May 15th."**
`[source: https://raw.githubusercontent.com/RooCodeInc/Roo-Code/main/README.md]` — the docs moved from
`docs.roocode.com` to `roocodeinc.github.io/Roo-Code/` and their index is stamped "Last updated on
**May 15, 2026**".

- **Skills.** `.agents/skills/{skill-name}/SKILL.md` and `~/.agents/skills/{skill-name}/SKILL.md`, plus
  `.roo/skills/` and `~/.roo/skills/`, plus mode-scoped `skills-{modeSlug}/` variants. Priority is an
  8-level list from "Project `.roo` mode-specific" (highest) to "Global `.agents` generic" (lowest).
  Frontmatter `name` and `description` required; names "1–64 characters, lowercase letters/numbers/hyphens
  only"; descriptions "1–1024 characters". `[source: https://roocodeinc.github.io/Roo-Code/features/skills/]`
- **Instructions.** `.roo/rules/`, `.roo/rules-{modeSlug}/`, fallback single files `.roorules` /
  `.roorules-{modeSlug}`; global `%USERPROFILE%\.roo\rules\`; `AGENTS.md` (or `AGENT.md`) loaded by default,
  disable with `"roo-cline.useAgentRules": false`. "The global rules directory location is fixed and cannot
  be customized". `[source: https://roocodeinc.github.io/Roo-Code/features/custom-instructions]`
- **Hooks.** **None found** — no hooks page, no `.roo/hooks/`, no hook column in the docs' comparison tables.
- **Worktrees.** Full feature page: worktrees get their own VS Code window; `.worktreeinclude` copies
  untracked files, and "Files must also be in `.gitignore` to be copied (intersection of both files)".
  `[source: https://roocodeinc.github.io/Roo-Code/features/worktrees]`

### 3.10 Kilo Code

- **Skills.** `~/.kilo/skills/` (`\Users\<yourUser>\.kilo\skills\` on Windows); `.kilo/skills/` in a project;
  "`.agents/skills/` — Open agent standard, loaded by default"; `.claude/skills/` when Claude Code
  compatibility is enabled. Config keys `skills.paths` and `skills.urls`. Kill switch
  `KILO_DISABLE_EXTERNAL_SKILLS=true`. `name` "**must match** the parent directory name". Skill bodies may
  embed shell via `` !`command` `` with a `KILO_DISABLE_SKILL_SHELL` kill switch.
  `[source: https://kilo.ai/docs/customize/skills]`
- **Instructions.** `instructions` key in `kilo.jsonc` (project) / `~/.config/kilo/kilo.jsonc` (global),
  e.g. `{"instructions": [".kilo/rules/formatting.md", ".kilo/rules/*.md"]}`; `.kilocode/rules/` still
  honoured. `AGENTS.md` > `AGENT.md` at project root; per-directory `AGENTS.md` are "**dynamically loaded**
  when the agent reads files in that directory". Priority table: agent prompt → project instructions →
  `AGENTS.md` → global instructions → skills on demand.
  `[source: https://kilo.ai/docs/customize/agents-md]`
- **Hooks.** Plugins in `~/.config/kilo/plugin/`, `.kilo/plugin/` (legacy `.kilocode/plugin/`); hooks
  include `config`, `event`, `tool`, `tool.execute.before`, `tool.execute.after`, `chat.message`,
  `permission.ask`, `command.execute.before`, `shell.env`, `experimental.*`. Blocking via `throw`. Kilo
  states "Upstream docs (behavior is identical to OpenCode)".
  `[source: https://kilo.ai/docs/automate/extending/plugins]`
- **Worktrees.** No user-facing worktree creation documented; plugin context exposes `worktree` = "Git
  worktree root for this session". `experimental_workspace` adaptors create folder/remote workspaces, not
  Git worktrees. **UNVERIFIED**.

### 3.11 Zed

- **Skills.** Only two roots, both `.agents`: global `~/.agents/skills/` and project-local
  `<worktree>/.agents/skills/`. "Each skill is a direct child of the skills root. Nesting skills inside
  subfolders is not supported." Project-local skills load "only from trusted worktrees". "If a global and a
  project-local skill share the same name, the project-local skill takes precedence." Frontmatter: `name`
  (required, lowercase/numbers/hyphens, max 64, "Should match the folder name") and `description`
  (required, "Keep it under 1024 bytes"), optional `disable-model-invocation`. Limits: "**50KB catalog
  budget**" for all names+descriptions, "**No remote registry**", "**Live reload**".
  `[source: https://zed.dev/docs/ai/skills]`
- **Instructions.** "Zed supports `AGENTS.md` as the primary instruction file". Personal
  `~/.config/zed/AGENTS.md`, or on Windows `%APPDATA%\Zed\AGENTS.md`. Project: "Zed uses the first matching
  file in this list: `.rules`, `.cursorrules`, `.windsurfrules`, `.clinerules`,
  `.github/copilot-instructions.md`, `AGENT.md`, `AGENTS.md`, `CLAUDE.md`, `GEMINI.md`". "Project
  instructions override personal `AGENTS.md` when they conflict." Note "Rules have been replaced by Skills
  and Instructions". `[source: https://zed.dev/docs/ai/instructions]`
- **Hooks.** No agent-lifecycle hook system. The only hook mechanism is **task hooks**: "The following hooks
  are currently supported: `create_worktree` — runs after Zed creates a new linked Git worktree … The task is
  spawned with `ZED_WORKTREE_ROOT` pointing at the newly created worktree and `ZED_MAIN_GIT_WORKTREE`
  pointing at the original repository's working directory." "Hook tasks are resolved from the same global and
  worktree-local `tasks.json` files as manually spawned tasks, and multiple tasks may register for the same
  hook; they all run when the hook fires." The docs' own example entry:

  ```json
  {
    "label": "copy .env into new worktree",
    "command": "cp",
    "args": ["$ZED_MAIN_GIT_WORKTREE/.env", "$ZED_WORKTREE_ROOT/.env"],
    "hooks": ["create_worktree"],
    "reveal": "no_focus",
    "hide": "on_success"
  }
  ```

  `[source: https://zed.dev/docs/tasks]`
- **Worktrees.** Yes: "New worktrees are created in a detached HEAD state", and "To automate setup steps
  whenever a new worktree is created, use a Task hook."
  `[source: https://zed.dev/docs/ai/parallel-agents]`

### 3.12 Amazon Q Developer

- **Rebrand first:** "The Q CLI has become the Kiro CLI." — `docs.aws.amazon.com/amazonq/latest/qdeveloper-ug/command-line.html`
  now only points at `https://kiro.dev/docs/cli`. The `aws.github.io/amazon-q-developer-cli/` docs describe
  themselves as "experimental, work in progress, and subject to change … they do not represent latest stable
  builds".
- **Skills.** No first-party page documents `SKILL.md` support for Q or the Q CLI. **UNVERIFIED / absent.**
- **Instructions.** "Project rules are defined in Markdown files in the project's `project-root`/.amazonq/rules`
  folder", used automatically as context; plain Markdown with no frontmatter.
  `[source: https://docs.aws.amazon.com/amazonq/latest/qdeveloper-ug/context-project-rules.html]`
  User-scope `~/.aws/amazonq/rules/` — **UNVERIFIED**.
- **Hooks.** A `hooks` field in the agent JSON (the filename minus `.json` is the agent name), with
  **5 events**: `agentSpawn`, `userPromptSubmit`, `preToolUse` ("Can block the tool use"), `postToolUse`,
  `stop`; each hook is `{ "command": ..., "matcher": ... }`.
  `[source: https://aws.github.io/amazon-q-developer-cli/agent-format.html]`
  The same page links a "[Hooks documentation](hooks.html)" page that **404s**, so the payload schema is
  undocumented. `~/.aws/amazonq/hooks.json` is **UNVERIFIED**.
- **Worktrees.** **UNVERIFIED.**

### 3.13 Aider

- **Instructions.** "create a small markdown file and include it in the chat … like `CONVENTIONS.md` …
  load the conventions file with `/read CONVENTIONS.md` or `aider --read CONVENTIONS.md`"; or persist it in
  `.aider.conf.yml` with `read: [CONVENTIONS.md, anotherfile.txt]`.
  `[source: https://aider.chat/docs/usage/conventions.html]`
- **Config.** "Aider will look for a this file in these locations: Your home directory. The root of your git
  repo. The current directory. If the files above exist, they will be loaded in that order. Files loaded
  last will take priority." `$XDG_CONFIG_HOME` is **not documented**. `.aiderignore` defaults to
  `.aiderignore` in the git root. `[source: https://aider.chat/docs/config/aider_conf.html]`
- **Skills / hooks / worktrees.** None documented — **UNVERIFIED**. The only hooks mentioned are git's own
  pre-commit hooks, bypassable with `--git-commit-verify`.

### 3.14 Continue

- **Instructions.** `.continue/rules/*.md` — "Create a folder called `.continue/rules` at the top level of
  your workspace"; "Ensure rules are in `.continue/rules/` (not `.continue/rule/`)"; "Rules files are loaded
  in lexicographical order". Frontmatter: `name` (required), `globs`, `regex`, `description`, `alwaysApply`
  (`true` = always included; `false` = included if globs match or the agent pulls it in; `undefined` =
  included if no globs or globs match). "Rules are not included in autocomplete or apply."
  `[source: https://docs.continue.dev/customize/deep-dives/rules]`
- **Skills / hooks / worktrees / AGENTS.md.** **UNVERIFIED** — no Skills or Hooks page exists in the docs
  navigation. `[source: https://docs.continue.dev/customize/rules]`

### 3.15 JetBrains Junie

Docs moved to `junie.jetbrains.com/docs/*.html`.

- **Skills.** Project `<projectRoot>/.junie/skills/<skill-name>/`; user `~/.junie/skills/<skill-name>/`
  (`%USERPROFILE%\.junie\skills\<skill-name>\` on Windows); **plus** "`.agents/skills/` directories:
  `<projectRoot>/.agents/skills/` (in a trusted project) and `~/.agents/skills/`". Disable all defaults with
  `--skill-default-locations false`; add dirs with `--skill-location`. Frontmatter: `name` required,
  `description` optional (falls back to the first body paragraph). Project skills beat user skills of the
  same name. Junie also *detects* `.cursor/skills/`, `.claude/skills/`, `.codex/skills/` and offers to
  import them. `[source: https://junie.jetbrains.com/docs/agent-skills.html]`
- **Instructions.** `.junie/AGENTS.md` (canonical, used exclusively if present) → `AGENTS.md` at project
  root combined with `.junie/playbook.md` and `.junie/rules/*.md` → `.junie/guidelines.md` (legacy). Global
  `~/.junie/AGENTS.md`; "Project-level guidelines always take precedence over global ones when they
  conflict". `[source: https://junie.jetbrains.com/docs/guidelines-and-memory.html]`
- **Hooks.** **None found** — no lifecycle hook feature or schema in any Junie page. Adjacent mechanisms:
  `~/.junie/allowlist.json`, `.aiignore`, `.junie/plans`.
- **Worktrees.** `/worktree` — "Junie creates a new Git worktree with a predefined name, such as
  `<project>-junie-wt-01` … as a sibling directory of your project"; "Worktree directories are created as
  siblings of the project directory, for example `../my-project-junie-wt-01`. Make sure the parent directory
  is writable." Uncommitted changes move with `git stash`. "Parallel sessions do not isolate files by
  themselves." `[source: https://junie.jetbrains.com/docs/junie-cli-worktrees.html]`

### 3.16 Google Antigravity

- **Skills.** "`<workspace-root>/.agents/skills/<skill-folder>/` — Workspace-specific";
  "`~/.gemini/config/skills/<skill-folder>/` — Global (all workspaces)"; "Antigravity now defaults to
  .agents/skills, but still maintains backward support for .agent/skills." Frontmatter: `description`
  required, `name` optional ("Defaults to the folder name if not provided").
  `[source: https://antigravity.google/docs/skills]`
  **DISCREPANCY:** the IDE page gives the global path as `~/.gemini/antigravity/skills/<skill-folder>/`
  while the 2.0 page gives `~/.gemini/config/skills/`. Both agree exactly on the workspace path.
  `[source: https://antigravity.google/docs/ide/skills]`
- **Instructions.** "Global rules live in `~/.gemini/GEMINI.md`"; "Workspace rules live in the `.agents/rules`
  folder of your workspace or git root" (`.agent/rules` legacy); activation modes Manual / Always On / Model
  Decision / Glob; "Rules files are limited to 12,000 characters each." `AGENTS.md` support is
  **UNVERIFIED**. `[source: https://antigravity.google/docs/rules-workflows]`
- **Hooks.** "Hooks are configured in a `hooks.json` file located in your customization directory (e.g.,
  `.agents/` in your workspace or `~/.gemini/config/`)." **5 events**: `PreToolUse`, `PostToolUse`,
  `PreInvocation`, `PostInvocation`, `Stop`. Top level is **name-keyed**, not event-keyed:

  ```json
  {
    "my-linter-hook": {
      "PostToolUse": [
        { "matcher": "run_command", "hooks": [ { "type": "command", "command": "./scripts/lint.sh", "timeout": 10 } ] }
      ]
    }
  }
  ```

  Handler fields: `type` (default `"command"`), `command` (required), `timeout` — "**Timeout in seconds.
  Defaults to `30`**". Hook-level `enabled` boolean. Payload: stdin JSON, camelCase (`conversationId`,
  `workspacePaths`, `transcriptPath`, `artifactDirectoryPath`, `modelName`).
  `[source: https://antigravity.google/docs/hooks]`
- **Worktrees.** Confirmed: "Projects natively support Git worktrees, allowing agents to operate in isolated
  background folders." Created via the per-conversation worktree selector: "**New Worktree Mode**: Creates a
  new Git worktree for the conversation." The location is **not documented — UNVERIFIED**.
  `[source: https://antigravity.google/docs/projects/]`

### 3.17 pi (`badlogic/pi-mono`, Mario Zechner)

- **Skills.** "Pi loads skills from: Global: `~/.pi/agent/skills/`, `~/.agents/skills/` — Project (only
  after the project is trusted): `.pi/skills/`, `.agents/skills/` in `cwd` and ancestor directories (up to
  git repo root, or filesystem root when not in a repo)". Discovery detail: "In `~/.agents/skills/` and
  project `.agents/skills/`, root `.md` files are ignored, but nested `.md` files in grouping folders are
  discovered when they declare skill frontmatter"; "In all skill locations, directories containing
  `SKILL.md` are discovered recursively". Frontmatter: `name` (max 64, lowercase/hyphens) and `description`
  (max 1024) required; `license`, `compatibility`, `metadata`, `allowed-tools`,
  `disable-model-invocation` optional. Pi deliberately deviates: "Pi does not require the name to match the
  parent directory. The Agent Skills standard does." `[source: https://raw.githubusercontent.com/badlogic/pi-mono/main/packages/coding-agent/docs/skills.md]`
- **Instructions.** "Pi loads `AGENTS.md` (or `CLAUDE.md`) at startup from: `~/.pi/agent/AGENTS.md`
  (global), parent directories (walking up from cwd), current directory"; `AGENTS.override.md` wins in its
  directory; `--no-context-files` disables.
  `[source: https://raw.githubusercontent.com/badlogic/pi-mono/main/packages/coding-agent/README.md]`
- **Hooks.** TypeScript extensions in `~/.pi/agent/extensions/*.ts` and `.pi/extensions/*.ts`; blocking via
  `return { block: true, reason: "Blocked by user" }` from e.g. `pi.on("tool_call", …)`. ~30 lifecycle
  events including `session_start`, `tool_call`, `tool_result`, `turn_end`, `session_shutdown`.
  No timeout documented. `[source: https://raw.githubusercontent.com/badlogic/pi-mono/main/packages/coding-agent/docs/extensions.md]`
- **Worktrees.** Not built in — "Git checkpointing and auto-commit" is listed as something an *extension*
  provides; philosophy section: "**No sub-agents.** There's many ways to do this." **UNVERIFIED** as a
  built-in feature.

---

## 4. Cross-harness comparison for hooking "a worktree was created"

berth's only automation need today is the moment a harness creates a worktree, so that
`berth adopt --setup` can register it. The relevant surfaces:

| Harness | Mechanism at worktree-creation time | Format | Notes |
| --- | --- | --- | --- |
| **Cursor** | `.cursor/worktrees.json` setup keys | JSON: `setup-worktree-unix` / `setup-worktree-windows` / `setup-worktree`, array of commands or script path | **berth already uses this.** Runs sequentially in the new worktree; `$ROOT_WORKTREE_PATH` available. Read by Agents Window, IDE and CLI. Worktrees live in `~/.cursor/worktrees/<reponame>/<name>`. |
| **Windsurf / Devin Desktop** | `post_setup_worktree` hook in `.windsurf/hooks.json` | JSON event array | "runs after each worktree is created and configured. It is executed inside the new worktree directory." `$ROOT_WORKSPACE_PATH` available. |
| **Claude Code** | `WorktreeCreate` hook event | JSON in `.claude/settings.json` | Fires for `--worktree`, `isolation: "worktree"` and background sessions. **A non-zero exit aborts worktree creation**, and the hook can *replace* `git worktree` entirely. `WorktreeRemove` is the teardown counterpart. |
| **Zed** | `create_worktree` task hook | JSON in `tasks.json` | Payload arrives as **environment variables** `ZED_WORKTREE_ROOT` / `ZED_MAIN_GIT_WORKTREE`, not stdin. |
| **Gemini CLI** | none — worktree setup is manual | — | Docs tell the user to initialise the environment themselves. |
| **Roo Code / Cline / Junie / Antigravity** | none documented | — | Worktrees exist; no documented repo-side setup hook. |
| **Codex / Copilot / opencode / Kilo / pi / Amazon Q / Aider / Continue** | no worktree creation documented | — | Nothing to hook. |

### 4b. Blocking and timeout — which hooks are unsafe for berth

`berth down` can wait for supervised processes, so a hook that must complete is in tension with harness
timeouts. Facts that matter:

| Harness | Session/turn-end budget | Safe for a long `berth down`? |
| --- | --- | --- |
| Claude Code | `SessionEnd` "shares a 1.5-second budget" (raisable to 60 s by setting a longer per-hook `timeout`) | **No** at default; 60 s is a hard ceiling |
| Codex CLI | `SessionEnd` and `Interrupt` "use `1` second by default and support up to `3` seconds" | **No** |
| Gemini CLI | `SessionEnd` is "**Best Effort**: The CLI **will not wait** for this hook to complete" | **No** — it is fire-and-forget |
| Cursor | `sessionEnd` is "a fire-and-forget hook … The response is logged but not used" | **No** |
| Copilot (VS Code) | no `SessionEnd` event exists | n/a |
| opencode | `session_shutdown`/`session.idle` events exist; no timeout documented | **UNVERIFIED** |
| Windsurf | blocking only for pre-hooks; post-hooks cannot block; no timeout documented | **UNVERIFIED** |
| Antigravity | `Stop` event, `timeout` default 30 s (seconds) | 30 s ceiling unless raised |
| pi | `session_shutdown` event; no timeout documented | **UNVERIFIED** |

Also relevant: **fail-open is the default in Cursor** ("hook failures (crash, timeout, invalid JSON) allow
the action through", `failClosed: false`), so a timed-out cleanup hook there does not merely fail silently —
it lets the action proceed.

---

## 5. The npx / CLI skill installers (all verified by execution)

Environment: Windows, `pwsh`, an isolated scratch directory under `%TEMP%` (never the repository checkout).
npm registry metadata was read via `npm view` and `https://api.npmjs.org/downloads/point/last-week/<pkg>`
on 2026-09-12; "weekly downloads" covers 2026-09-04 → 2026-09-10.

### 5.1 `skills` — the real one (Vercel Labs)

| Field | Value |
| --- | --- |
| npm package | `skills` |
| Version | `1.5.25` (dist-tag `latest`; also `snapshot`), last modified **2026-09-08** (exec) |
| Weekly downloads | **5,228,357** (exec) |
| Repository | https://github.com/vercel-labs/skills |
| Description | "The open agent skills ecosystem" |

Observed `npx --yes skills --help` (verbatim excerpt; exit 0):

```text
Usage: skills <command> [options]

Manage Skills:
  add <package>        Add a skill package (alias: a)
                       e.g. vercel-labs/agent-skills
                            https://github.com/vercel-labs/agent-skills
  use <package>@<skill>  Generate a prompt for using one skill without installing it
  remove [skills]      Remove installed skills
  list, ls             List installed skills
  find [query]         Search for skills interactively

Updates:
  update [skills...]   Update skills to latest versions (alias: upgrade)

Project:
  experimental_install Restore skills from skills-lock.json
  init [name]          Initialize a skill (creates <name>/SKILL.md or ./SKILL.md)
  experimental_sync    Sync skills from node_modules into agent directories

Add Options:
  -g, --global           Install skill globally (user-level) instead of project-level
  -a, --agent <agents>   Specify agents to install to (use '*' for all agents)
  -s, --skill <skills>   Specify skill names to install (use '*' for all skills)
  -l, --list             List available skills in the repository without installing
  -y, --yes              Skip confirmation prompts
  --copy                 Copy files instead of symlinking to agent directories
  --all                  Shorthand for --skill '*' --agent '*' -y
  --full-depth           Search all subdirectories even when a root SKILL.md exists

Installation Scope
  | Project | (default) | ./<agent>/skills/ |
  | Global  | -g        | ~/<agent>/skills/ |
```

README (fetched from `raw.githubusercontent.com/vercel-labs/skills/main/README.md`) documents the source
formats (`owner/repo`, full GitHub URL, "Direct path to a skill in a repo", GitLab, any git URL, local path),
`--global`, `--agent`, `--skill`, `--list`, `--copy`, `--all`, `--subagent`, `--full-depth`, and the
discovery locations it walks (`.agents/skills/`, `skills/`, `.claude/skills/`, `skills/.curated/`, …).

**Does it work for berth today? Yes — verified end to end (exec).**

`npx --yes skills add Mrjwj34/berth --list`:

```text
◇  Source: https://github.com/Mrjwj34/berth.git
◇  Repository cloned
◇  Found 1 skill

◇  Available Skills
│    berth
│      Manage independent local agent workspaces, explicit native/container runtimes, managed processes and safe cleanup. Use for parallel task worktrees, port allocation, starting or inspecting task services, running tests inside a workspace, or reclaiming workspace state.
└  Use --skill <name> to install specific skills
```

`npx --yes skills add Mrjwj34/berth --skill berth -a claude-code -a cursor --copy -y`:

```text
◇  Installation Summary
│  ~\AppData\Local\Temp\skills-probe\.agents\skills\berth
│    copy → Claude Code, Cursor
◇  Installation complete

◇  Installed 1 skill
│  ✓ berth (copied)
│    → ~\AppData\Local\Temp\skills-probe\.claude\skills\berth
│    → ~\AppData\Local\Temp\skills-probe\.agents\skills\berth
```

Resulting files (exec): `.agents/skills/berth/SKILL.md`, `.agents/skills/berth/references/{berth-yaml.md,recovery.md}`,
`.claude/skills/berth/…` (same), and a new `skills-lock.json` in the directory root.

Interpretation for berth:

- It finds berth's skill at `.agents/skills/berth/SKILL.md` with no extra configuration, and it installs
  per-agent, per-scope (`-g` for global), and honours `--skill` selection.
- It **only installs the skill text**. There is no npm package for the berth binary, so `npx skills add`
  cannot install berth itself; it is an alternative *distribution channel for the skill*, not a substitute
  for `berth init`.
- It writes a `skills-lock.json` into the target directory, which is a new repo artifact berth does not
  currently create.
- `--full-depth` exists because a root `SKILL.md` otherwise shadows nested ones.

### 5.2 `add-skill` — deprecated, same repo

`npm view` (exec): version `2.0.0`, description **"DEPRECATED: Use 'npx skills add' instead"**, repository
`git+https://github.com/vercel-labs/skills.git`, 618 weekly downloads, last modified 2026-01-26.
Do not recommend it; it redirects users to `skills`.

### 5.3 `skills-cli` — real but tiny

`npm view` (exec): version `0.1.3`, "NPM for AI agent skills - install, manage, and publish skills for
Claude Code, Codex, and more", repository `git+https://github.com/brunogalvao/claude-skills-directory.git`,
**139 weekly downloads**, last modified 2026-01-04. Real, but not a distribution channel berth should
depend on (single-maintainer, ~4 orders of magnitude less usage than `skills`).

### 5.4 Names that do **not** exist / are not installers

| Candidate | Result (exec) |
| --- | --- |
| `@skills/cli` | **404 — not in the registry.** `npm error code E404 … '@skills/cli@*' is not in this registry.` |
| `@vercel/skills` | **404 — not in the registry.** (The Vercel package is the unscoped `skills`.) |
| `agent-skills` | Exists (v1.0.4) but is "Node.js wrapper for agent-skills-mcp", 48 weekly downloads — not a skill installer from a GitHub repo. |
| `claude-skills` | Exists (v1.0.2) but is a Japanese-language personal skill *collection*, 13 weekly downloads — not an installer. |

### 5.5 `gh skill` — the GitHub CLI installer (preview)

Documented at https://cli.github.com/manual/gh_skill and confirmed locally by running it (exec):

```text
Install and manage agent skills from GitHub repositories.
Working with agent skills in the GitHub CLI is in preview and subject to change without notice.
USAGE
  gh skill <command> [flags]
AVAILABLE COMMANDS
  install:       Install agent skills from a GitHub repository (preview)
  list:          List installed skills (preview)
  preview:       Preview a skill from a GitHub repository (preview)
  publish:       Validate and publish skills to a GitHub repository (preview)
  search:        Search for skills across GitHub (preview)
  update:        Update installed skills to their latest versions (preview)
EXAMPLES
  $ gh skill install github/awesome-copilot documentation-writer
```

GitHub's docs add that it targets agents via `--agent`/`--scope` and that "By default, skills are installed
for Copilot at project scope".
`[source: https://docs.github.com/en/copilot/how-tos/copilot-on-github/customize-copilot/customize-cloud-agent/add-skills]`

---

## 6. Additional mainstream harnesses (evidence-based)

Beyond the 18 requested, these are the harnesses worth considering, with the evidence that they are
mainstream. Weekly npm downloads measured 2026-09-12 (exec):

| Harness | Evidence | `.agents/skills`? |
| --- | --- | --- |
| **Amp** (Sourcegraph) | `@sourcegraph/amp` — 15,239 weekly downloads (exec); first-party docs at https://ampcode.com/docs with a "Customize → Skills" section and "Global Plugins & Skills" | Project `.agents/skills/`, global `~/.config/agents/skills/` per the `skills` CLI table (**secondary source — UNVERIFIED**) |
| **Qwen Code** | `@qwen-code/qwen-code` — 48,403 weekly downloads (exec); first-party docs at https://qwenlm.github.io/qwen-code-docs/ | Not verified by me |
| **Crush** (Charm) | `@charmland/crush` — 4,821 weekly downloads (exec); skills documented in its repo README | Not verified by me |
| **Kiro CLI** | The Q Developer CLI's successor: "The Q CLI has become the Kiro CLI." `[source: https://docs.aws.amazon.com/amazonq/latest/qdeveloper-ug/command-line.html]` | `.kiro/skills/` and `~/.kiro/skills/` per the Kiro docs referenced by the `skills` CLI; also a `skill://` resource form in custom agents |
| **Droid** (Factory), **Warp**, **Dexto**, **Kimi Code CLI**, **Loaf**, **Sarvam Code**, **OpenClaw**, **Replit**, **Universal**, **Deep Agents**, **Firebender**, **PromptScript** | Listed as `--agent` targets by the `skills` CLI, whose README maps most of them to `.agents/skills/` `[source: https://raw.githubusercontent.com/vercel-labs/skills/main/README.md]` — **secondary source only** | Most map to `.agents/skills/`; not verified against their own docs |

Also measured for scale (exec): `@anthropic-ai/claude-code` 11,400,831 weekly downloads; `@openai/codex`
17,360,262; `opencode-ai` 1,738,770; `@google/gemini-cli` 312,297; `cline` 36,931; `@continuedev/cli` 2,723.

---

## 7. Recommendations for berth

### 7.1 The single-install design is sound — keep it, but stop calling it universal

`berth init` writing only `<repo>/.agents/skills/berth/SKILL.md` reaches **12 harnesses** with no
duplication: Codex, Cursor, Copilot (VS Code + cloud), Gemini CLI, opencode, Windsurf/Devin Desktop, Roo
Code, Kilo Code, Zed, Junie, Antigravity, pi. The README's four-harness claim understates it.

The design fails for exactly three groups, and each needs a decision rather than a default:

1. **Claude Code** does not read `.agents/skills` and does not read `AGENTS.md`. It needs
   `.claude/skills/berth/SKILL.md` (project) and/or `~/.claude/skills/berth/SKILL.md`.
2. **Cline** does not read `.agents/skills`; it needs `.cline/skills/berth/SKILL.md`.
3. **Aider, Continue, Amazon Q** have no skills mechanism at all. For them the only option is a rules/
   instructions file — which is a different artifact with different semantics, so it should be a separate,
   explicitly opt-in adapter rather than a "shape" of the skill install.

### 7.2 Concrete matrix — what berth should write

| Harness | Project scope | User/global scope | Mechanism | Default? |
| --- | --- | --- | --- | --- |
| Cursor, Codex, Copilot, Gemini CLI, opencode, Windsurf, Roo, Kilo, Zed, Junie, Antigravity, pi | `.agents/skills/berth/SKILL.md` (+ `references/`) | `~/.agents/skills/berth/SKILL.md` | one canonical copy | **Default (project)**, global opt-in |
| Claude Code | `.claude/skills/berth/SKILL.md` | `~/.claude/skills/berth/SKILL.md` | copy or symlink of the same body | opt-in per harness, **but prompt for it** — it is the largest installed base (`@anthropic-ai/claude-code`, 11.4 M weekly downloads) |
| Cline | `.cline/skills/berth/SKILL.md` | `~/.cline/skills/berth/SKILL.md` | copy | opt-in |
| Roo Code | covered by `.agents/skills` | covered | — | **Do not** target while the extension is shut down |
| Aider | `CONVENTIONS.md` reference is a *user* file; do not write it | — | `--read` is user-driven | **Do not support** |
| Continue | `.continue/rules/berth.md` | — | rules file, not a skill | opt-in, if at all |
| Amazon Q | `.amazonq/rules/berth.md` | — | rules file; CLI is now Kiro | **Do not support** |
| Zed | covered by `<worktree>/.agents/skills/` | covered by `~/.agents/skills/` | — | note: Zed must be *trusted* and only reads `.agents/skills`, so `berth init` already covers it |
| pi / Antigravity / Gemini CLI / Junie | covered | covered | — | note trust gates: pi and Junie only load project skills in a **trusted** project; Gemini CLI asks for consent on activation |

Skill body constraints to respect (all first-party):

- Keep the frontmatter to the Agent Skills spec's portable set — `name`, `description`, `license`,
  `compatibility`, `metadata`, `allowed-tools` — because the spec-conformant readers are the majority, and
  Claude Code **hard-errors** on out-of-spec keys for claude.ai upload/`package_skill.py`. berth's current
  `SKILL.md` (only `name` + `description`) is already within spec.
- `name` must be `berth` and, for the strict readers (VS Code Copilot, Kilo, Cline, Agent Skills spec), must
  match the directory name. berth complies.
- Keep the body under 500 lines (Claude Code and Zed both advise this); berth already pushes detail into
  `references/`, which all readers load on demand.
- Zed adds a **50 KB catalog budget** over all skill descriptions, and Codex caps the skills list at 2 % of
  the context window — long `description` text is a shared, scarce resource. berth's current description is
  ~250 characters, which is reasonable.

### 7.3 Worktree hooks — what to install, and what to refuse

**Cursor (already implemented, keep as default).** `.cursor/worktrees.json` is the only mechanism in the
whole matrix that is a *file-based, declarative* worktree setup step, read by Agents Window + IDE + CLI, and
berth's adapter matches the documented schema exactly. Two things to fix:

- The merge in `internal/app/hooks_install.go` replaces the three keys wholesale. Cursor merges *hooks* by
  priority, but `worktrees.json` has no documented merge semantics for the `setup-worktree*` keys, and the
  docs' own examples put unrelated commands (`npm ci`, `cp $ROOT_WORKTREE_PATH/.env .env`) in the same
  array. Overwriting a user's existing array silently deletes their setup steps. Append-if-absent is the
  safe strategy for the array form.
- Document that the IDE wording is ambiguous: the doc says the "UI-native worktrees feature … is only
  available in the Agents Window. In the IDE, use the Worktree Skills commands below", while the same page
  says the file is read "when it creates a worktree in the Agents Window, the IDE, or the Cursor CLI".

**Windsurf `post_setup_worktree`** is the closest analogue to Cursor's mechanism and is worth a second
adapter: it runs inside the new worktree with `$ROOT_WORKSPACE_PATH` set, and its payload includes
`worktree_path` and `root_workspace_path`. Merge into `.windsurf/hooks.json` by appending to the
`post_setup_worktree` array — Windsurf documents additive merge across system/user/workspace, so an entry
added by berth does not clobber others.

**Claude Code `WorktreeCreate` — implement only with care.** It is the most dangerous hook in the matrix for
a tool like berth: "any non-zero exit code from `WorktreeCreate` aborts worktree creation", and a
`WorktreeCreate` hook can replace `git worktree` entirely. `berth adopt --setup` failing (a port conflict, a
locked registry) would therefore silently prevent Claude from creating the worktree at all. If berth installs
it, it must exit 0 even when adopting fails, print the failure on stderr, and let `berth gc` clean up.

**Zed `create_worktree`** is viable but different in shape: it is a *task* in `tasks.json` receiving
`ZED_WORKTREE_ROOT` and `ZED_MAIN_GIT_WORKTREE` as environment variables, not stdin JSON. It also lives in
the same file as the user's normal tasks, so it needs a JSON array append (or a new hook registration)
rather than an object merge.

### 7.4 Hooks that are unsafe for berth's semantics (do not install)

| Hook | Why it is unsafe |
| --- | --- |
| Claude Code `SessionEnd` | 1.5 s shared budget, raisable only to 60 s. `berth down` waits for supervised processes; it would be cancelled and its output discarded ("a timed-out `command` hook doesn't block the tool call"). |
| Codex `SessionEnd` / `Interrupt` | 1 s by default, max 3 s. Same problem, worse. |
| Gemini CLI `SessionEnd` | "The CLI **will not wait** for this hook to complete" — it cannot be used for cleanup by construction. |
| Cursor `sessionEnd` | Fire-and-forget; "The response is logged but not used". Also note Cursor hooks are **fail-open by default**, so a timeout there means the *action proceeds*, not that berth's cleanup ran. |
| Copilot (VS Code) end-of-session | No `SessionEnd` event exists at all; the nearest is `Stop`, which is per-turn, and the default `timeout` is 30 s. |
| Any `PreToolUse`-style hook that shells out to berth | These sit on the critical path of every tool call ("Gemini CLI waits for all matching hooks to complete"; Claude Code's agent waits up to the timeout; Codex "waits for a command hook to finish before continuing the operation"). A per-tool berth call would add latency to every action and a stall would be blamed on berth. |

Recommended posture, matching the README's existing rationale: berth installs **only** worktree-adoption
hooks; it installs no session-end cleanup and nothing on the per-tool path. `berth down` / `berth done`
stay explicit user/agent commands, and `berth gc` reclaims abandoned workspaces — exactly the argument the
README already makes.

### 7.5 Merge strategy, in order of safety

1. **Prefer a file berth owns entirely.** `.cursor/worktrees.json` (Cursor) and `.agents/skills/**` are
   berth-owned artifacts; no merge needed.
2. **Append to an array under a named key.** Windsurf `post_setup_worktree`, Cursor's `setup-worktree`
   arrays: read → decode → append if no berth entry → write. This preserves user entries, which the
   harness documents as additive anyway.
3. **Add a sibling key inside the harness's existing `hooks` object.** Claude Code documents this discipline
   verbatim: "add `Notification` as a sibling of the existing event keys rather than replacing the whole
   object". Preserve unknown keys: decode into a generic map, mutate, re-encode — berth's current Cursor
   merge already does this, and the same technique must be used for `.claude/settings.json`.
4. **Never assume atomicity.** No harness documents a locking, backup or partial-write contract for
   third-party hook installation. Claude Code's failure mode is instructive: invalid JSON is a **Settings
   Error** that blocks the session, while a bad individual entry is only a **Settings Warning**. Write to a
   temp file and rename, and never write a file berth cannot fully parse first.

### 7.6 Harnesses not worth supporting, with reasons

- **Aider** — no skills, no hooks, no worktrees; instructions are a user-chosen `--read` file. Nothing to
  install that would not be presumptuous.
- **Amazon Q Developer** — the CLI is now Kiro CLI, its technical docs self-describe as "experimental, work
  in progress … they do not represent latest stable builds", there is no documented skills support, and its
  own hooks documentation link 404s. Support Kiro CLI later if it stabilises, not Q.
- **Continue** — no skills support; the only surface is `.continue/rules`, a different artifact class.
- **Roo Code** — the extension was shut down on 2026-05-15; `.agents/skills` support is therefore moot.
- **JetBrains Junie** — it *does* read `.agents/skills`, so it costs nothing to support, but it has no hook
  surface at all, so berth can never adopt its worktrees automatically. Document that limitation rather than
  implying integration.
- **GitHub Copilot cloud agent / Copilot code review** — the environment is an ephemeral Actions runner
  configured by `.github/workflows/copilot-setup-steps.yml`; there is no worktree, so berth's model does not
  apply. Its skills support is still worth having.
- **Codex, opencode, Kilo, pi, Gemini CLI, Antigravity** — supported for skills only, because none documents
  a worktree-creation configuration surface berth can hook. This is a documentation gap, not necessarily a
  product gap; re-check Gemini CLI (experimental worktrees) and Antigravity (New Worktree Mode) later.

---

## 8. Unverified / could not confirm

Consolidated. Everything here should be checked before it is coded against.

**General**

1. **Documented negatives.** No harness says "we do not read `.agents/skills`". The "No" rows in §2 rest on
   the absence of the path from tables/documentation sets that present themselves as complete.
2. **`~/.config/github-copilot/skills`** — no doc found; not a documented Copilot location.
3. **Programmatic hook installation** — no harness documents an API, a read-modify-write contract, atomic
   writes, backups, or unknown-key preservation for a third party adding a hook entry.

**Per harness**

4. **Claude Code** — the `name` frontmatter field's length limit and character class are not stated (only
   the SDK's `skills` allowlist option documents rejections). Cloud/web sessions are documented as
   Anthropic-managed VMs, not worktrees; whether they create one is unknown. Agent-team teammates are
   explicitly **not** worktree-isolated.
5. **Codex CLI** — the literal `.agents` string was not found in the source tree (unauthenticated GitHub
   code search returned HTTP 403 rate limits); the `.agents/skills` claim rests on the official doc table.
   `.codex/skills` and `.codex/prompts` as skill locations: not documented, treated as absent. `notify`'s
   timeout and blocking behaviour are undocumented. No worktree support found.
6. **Cursor** — the `timeout` default is documented only as "platform default", with no value. No page states
   literally that worktree setup completes **before** the agent starts. No page documents an IDE *worktree
   toggle* (the toggle control is the Agents Window picker; the IDE has `/worktree` and `/best-of-n` slash
   commands instead). The CLI binary name in current docs is `agent`; `cursor-agent` appears nowhere, so it
   is **UNVERIFIED** as a documented name. The docs' `preToolUse` input example is not literally valid JSON
   as printed (it contains `"allow" | "deny"` unions), so treat it as a schema illustration.
7. **Copilot** — the exact agent-instructions filename on GitHub.com is unknown (the docs' reusable snippet
   did not render; only `CLAUDE.md`/`GEMINI.md` are literal on that page). The per-row filenames in the
   custom-instructions support matrix did not render either. No worktree mechanism documented.
8. **Gemini CLI** — `$GEMINI_CONFIG_DIR` is not documented (the analogue is `GEMINI_CLI_HOME`). There is no
   documented `gemini hooks` CLI subcommand (only `/hooks` slash commands). The docs disagree on
   `/memory refresh` vs `/memory show`.
9. **opencode** — `chat.message`, `chat.params`, `chat.headers`, `permission.ask`, `command.execute.before`
   and `tool.definition` are **not** in opencode's own plugin docs (they appear only in Kilo Code's, which
   claims identical behaviour). `.opencode/plugin/*.ts` (singular) is not documented by opencode.
10. **Windsurf** — no hook timeout field is documented. Auto-generated memories are legacy-Cascade-only.
11. **Cline** — the legacy hook set (`TaskStart`, `TaskResume`, `TaskCancel`, `TaskComplete`, `PreToolUse`,
    `PostToolUse`, `UserPromptSubmit`, `PreCompact`, `~/Documents/Cline/Hooks/`, `.clinerules/hooks/`,
    `{"cancel": true, …}`) is absent from the current docs. `.clinerules/workflows/` and a single-file
    `.clinerules` are likewise unverified.
12. **Roo Code** — hooks: none found. Docs frozen at 2026-05-15.
13. **Kilo Code** — Git worktree creation: not documented. Some Kilo doc endpoints advertised in its
    `llms.txt` return 404.
14. **Zed** — whether Zed creates worktrees in any surface beyond the Git panel / parallel agents, and
    whether its `.agents/skills` loader has any global-scope variant beyond `~/.agents/skills/`, was not
    exhaustively checked. Zed's docs state skills must be flat children of the root, so nested category
    folders (which Cursor supports) will not load in Zed.
15. **Amazon Q Developer** — user-scope rules path, skills support, hook payload schema (its linked hooks
    page 404s), and worktrees are all unconfirmed. `docs.aws.amazon.com` returns HTTP 200 with a bare shell
    for nonexistent paths, so a 200 there is not evidence a page exists.
16. **Aider / Continue** — absence of skills, hooks and worktrees is based on their documentation sets, not
    on statements of absence.
17. **Junie** — no lifecycle hook feature found; whether one exists undocumented was not exhaustively
    probed.
18. **Antigravity** — worktree location is undocumented. The global skills root differs between the 2.0 page
    (`~/.gemini/config/skills/`) and the IDE page (`~/.gemini/antigravity/skills/`); no page reconciles
    them. `AGENTS.md` support is unconfirmed. The IDE hooks page (`/docs/ide/hooks/`) was seen in the
    navigation but not read.
19. **pi** — star count unknown (no badge in the README). The repo's own `docs/extensions.md` links tool
    sources to `github.com/earendil-works/pi`, a different org path from the `badlogic/pi-mono` checkout
    read; the naming is inconsistent upstream. No built-in worktree creation.
20. **Amp, Qwen Code, Crush, Kiro CLI and the rest of §6** — mainstream status is evidenced, but their
    `.agents/skills` mapping comes from the `skills` CLI's README (secondary source) unless noted; they were
    not checked against their own docs.

**Environment notes that affected this research**

- `developers.openai.com/codex/*` and `docs.anthropic.com`/`docs.claude.com` 302 cross-origin (to
  `learn.chatgpt.com` and `code.claude.com` respectively); `docs.windsurf.com/*` now 302s to
  `docs.devin.ai`; `kilocode.ai/docs` 302s to `kilo.ai`; `jetbrains.com/help/junie/*` 302s to
  `junie.jetbrains.com`. Citations use the URL that actually served the content.
- Unauthenticated GitHub API calls returned **HTTP 403 rate limits** during this session, so repository
  source trees could not be enumerated and star counts could not be read.
- `web_search` was unavailable; navigation came from sitemaps, doc indexes and `llms.txt` files.
