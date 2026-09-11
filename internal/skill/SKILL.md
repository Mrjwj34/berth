---
name: lane
description: Manage isolated local Agent workspaces with git worktrees, unique ports, private data dirs, and project-declared processes.
---

# lane

Project-agnostic, harness-agnostic local Agent workspace CLI. Isolation primitives only: Git worktree, named port block (`LANE_PORT_<NAME>` published on the host), private `LANE_DATA_DIR`, project-declared processes. Identity is the directory path. lane does not know Postgres, MySQL, Redis, or any other service. Do not change a project so it can run under lane.

## 1. When to use

Use lane when a task needs an isolated local workspace: unique ports, a private data dir, and optional project processes — without a VM as the isolation layer.

Do not use raw `git worktree`, process-compose, or hand-picked ports. Use this skill and the `lane` CLI.

## 2. Project onboarding

When the repo has no `lane.yaml` (or it is incomplete), onboard it:

1. **Read how the project starts.** Find the app, tests, and every datastore. Note every listen port and every on-disk path. Leave those ports and commands as the project wrote them.
2. **Do not rewrite the app.** Set `isolate: net`. Processes keep binding `:8080` / `5173` / whatever they already use, stay on the host network, and can still reach the public internet. Intra-workspace `localhost:8080` keeps working. lane rewrites those bind/connect calls to unique host ports (`lane ports --json`). Optional `ports.api.listen: 8080` only names a host port as `LANE_PORT_API`. Only if a process already accepts `--port` / `$PORT` may you skip isolation and pass `$LANE_PORT_API`.
3. **Private data only when the project shares a path across checkouts.** Point those paths at `$LANE_DATA_DIR`. Do not invent env-driven ports just so lane can start.
4. **Write `lane.yaml`** at the repo root. Declare `ports`, `env` (URLs that *other* tools need on the host), `hooks.setup`, and `processes`. Recipes in `docs/recipes/` are ordinary processes — not first-class service types.
5. **Self-test:** `lane new smoke --up`. Confirm readiness from JSON status/ports. Then `lane done smoke`.
6. **Commit** `lane.yaml` (and `.worktreeinclude` / `env_file` if used).

Always-injected env: `LANE_DATA_DIR`, `LANE_WORKSPACE` (abs worktree path), `LANE_SLUG`, `LANE_ROOT` (main repo path), `LANE_BRANCH`, `LANE_PORT_<NAME>` (host-published port), and `LANE_LISTEN_<NAME>` when `listen:` is set.

Default data dir: `<worktree>/.lane/data`. Branch created as `lane/<slug>` from `base` (default `main`). Worktree default path: sibling `<parent>/<reponame>.lanes/<slug>`.

Optional config: `.worktreeinclude` (one relative path per line, `#` comments) copies extra files from the main repo; `copy_dirs` fast-clones dirs like `node_modules`; `env_file` writes a managed block between `# BEGIN LANE` / `# END LANE`.

## 3. Daily lifecycle

One task, one workspace.

1. `lane new <slug>` (add `--up` to start processes; `--base <branch>` if not `main`).
2. **Work only in the printed path.** Do not edit the main checkout.
3. `lane up` / `lane status --json` / `lane logs [proc]` / `lane ports --json` / `lane run -- <cmd>` as needed.
4. `lane done [slug]` when finished. `lane gc` reclaims leftovers.

`lane` (no args) is the global overview. `lane attach [slug]` enters an existing workspace. `lane reset` stops processes, wipes `$LANE_DATA_DIR`, and re-runs `hooks.setup`.

## 4. Iron rules

- **Never bypass lane.** No raw `git worktree add/remove`, no invoking process-compose, no ad-hoc ports. process-compose is auto-fetched (pinned v1.122.0) into `$LANE_HOME/bin`; agents never run it.
- **On failure:** run `lane doctor` (then `lane doctor --fix` if appropriate). Report the diagnosis. Do not invent a workaround.
- **Never `--force`** without explicit user authorization.
- **Never print secrets.** Connection strings and env files stay off the transcript.

## 5. Harness

Core is **CLI + `lane.yaml` + this skill**. Cursor `worktrees.json` and Claude Code hooks are optional adapters (`lane hook install [cursor|claude|all]`).

If Cursor or Claude already created a worktree, `cd` into that directory and run `lane adopt` (or `lane adopt --setup`). Do not create a second workspace for the same task.

## 6. Actionable failures

Errors name the cause and the next command. Match that style; do not swallow them.

- `worktree is dirty. Commit changes or use --force`
- Missing `lane.yaml`: run `lane init`, then complete ports/env/processes.
- Unhealthy process or port: `lane status --json`, `lane logs [proc]`, `lane doctor`.

Query commands (`ls`, `status`, `ports`, `doctor`) accept `--json`. Prefer JSON when parsing.

## Command reference

| Command | Purpose |
| :--- | :--- |
| `lane` | Global overview |
| `lane init` | Commented `lane.yaml` + skill install |
| `lane skill install` | Install this skill into agent skill dirs |
| `lane hook install [cursor\|claude\|all]` | Optional Cursor/Claude adapters |
| `lane new <slug> [--up] [--base <branch>]` | Create workspace; prints the worktree path |
| `lane attach [slug]` | Enter an existing workspace |
| `lane adopt [--setup]` | Register an existing worktree from cwd |
| `lane ls\|status\|ports [--json]` | List / current status / port map |
| `lane up\|down` | Start / stop project processes |
| `lane logs [proc]` | Process logs |
| `lane run -- <cmd>` | Run a command with `LANE_*` injected |
| `lane reset` | Wipe data dir and re-run setup |
| `lane done [slug] [--force]` | Tear down a workspace |
| `lane gc [--dry-run]` | Reclaim vanished/idle/merged/excess workspaces |
| `lane doctor [--fix]` | Diagnose (and optionally repair) the toolchain |
| `lane open [port-name]` | Open a named port in the browser |

State file: `$LANE_HOME/state.json` (default `~/.lane/state.json`). Tests set `LANE_HOME`. GC: vanished dirs reclaim ports; `idle_stop_hours` stops processes; `remove_after_days` removes merged-branch workspaces; `max_workspaces` evicts the oldest clean stopped workspaces.
