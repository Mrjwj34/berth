---
name: lane
description: Manage independent local agent workspaces, explicit native/container runtimes, managed processes and safe cleanup. Use for parallel task worktrees, port allocation, starting or inspecting task services, running tests inside a workspace, or reclaiming workspace state.
---

# lane

lane gives each task its own Git worktree, private data directory, reserved ports
and supervised processes, so parallel agents stop colliding. It is a workspace
manager, not an adversarial security sandbox and not a Git replacement. Never
replace lane lifecycle operations with raw `rm -rf`, `git worktree remove`, or
guessed PID/port cleanup.

## Onboard

Read the project's startup scripts, toolchains, storage paths and listen ports
before writing configuration, then choose the runtime deliberately:

- `native` (default) for programs that accept a port and a data path through
  configuration or flags. Fastest, no container overhead.
- `container` for unchanged hardcoded listen ports or a uniform Linux
  environment. Requires `runtime.backend: container`, `runtime.engine:
  docker|podman`, an existing `runtime.image`, and `listen.<port>` for every
  declared port.

Do not attempt preload/seccomp injection. If the Linux engine or the prepared
image is missing, report it and stop; never silently run the service on the host
instead.

Commit `lane.yaml` before creating a workspace. Each workspace loads the
configuration from its own branch, so uncommitted edits never reach a workspace
created from it.

A container image must include the project toolchain, bash, sh, sleep, socat,
git, python3 and process-compose v1.122.0; `runtime/Dockerfile` is the reference.
Build or install dependencies once per image rather than on every run.

Keep mutable state in `$LANE_DATA_DIR`. `copy_dirs` copies writable dependency
trees with CoW where the filesystem supports it and an independent copy
otherwise; hardlinks are never used. Treat external symlinks and absolute shared
storage explicitly.

## Execute

| Command | Purpose |
| --- | --- |
| `lane new <slug> [--base <ref>] [--up] [--print-path] [--json]` | Create a workspace and its branch-local setup; `--up` also starts processes |
| `lane ls [--json]` | List workspaces with their phase, running state and ports |
| `lane status [<slug>] [--json]` | Processes, readiness and ports of a workspace |
| `lane ports [<slug>] [--json]` | Allocated host ports and container listen mappings |
| `lane plan [<slug>]` | Print the execution contract as JSON without starting anything |
| `lane up [<slug>]` / `lane down [<slug>]` | Start / stop workspace processes, preserving data |
| `lane logs <proc> [<slug>]` | Logs of one managed process |
| `lane run [--] <cmd> [args...]` | Run one command inside the workspace runtime and environment |
| `lane attach [<slug>] [--json]` | Print path, branch and shell exports for the workspace |
| `lane open [<port-name>]` | Open a workspace port's host URL in the browser |
| `lane reset [<slug>]` | Wipe `$LANE_DATA_DIR` and rerun setup |
| `lane done [<slug>]` | Tear a workspace down once its work is preserved |
| `lane gc [--dry-run] [--json]` | Reclaim vanished, idle, merged or excess workspaces |
| `lane doctor [--fix] [--json]` | Check git, engine, image, process-compose and state |

### Where a workspace is

- Path: `<repo>.lanes/<slug>` beside the repository by default, or
  `<worktree_root>/<slug>` when `worktree_root` is set (relative values resolve
  inside the repository). Branch: `lane/<slug>`.
- Slug rules: lowercase ASCII letters, digits, `-` and `_` only, no slashes.
- The primary checkout is never a workspace (`lane adopt` refuses it). `lane run`
  and `lane open` accept no slug, so they resolve the workspace from the current
  directory; the commands that do take `[<slug>]` also work from anywhere with an
  unambiguous slug. Run slug-less commands from inside the path printed by `lane
  new` (its last stdout line, or `path` in `--json`); elsewhere they fail with
  `workspace not found`.

`lane run` joins the same runtime and environment as services and hooks, passes
argv verbatim without reparsing, and propagates the exit code. Request a shell
explicitly: `lane run -- sh -c '<script>'` (Linux/container) or the matching
native shell on Windows. Do not assume a script is portable between `sh` and
Windows `cmd`. Cancelling a container `lane run` stops that workspace's runtime,
including its sibling services.

### Environment contract

- `LANE_PORT_<NAME>` is the port inside the execution context;
  `LANE_HOST_PORT_<NAME>` is the host publication to hand to browsers and other
  host tools. Container publications are TCP over IPv4 loopback.
- `plan.gateway_ports` are internal forwarder ports; application listeners must
  not use those values. Undeclared internal ports are isolated but not
  discovered or published.
- Identity: `LANE_WORKSPACE`, `LANE_ROOT`, `LANE_DATA_DIR`, `LANE_SLUG`,
  `LANE_BRANCH`. Container mode adds `GIT_DIR`, `GIT_WORK_TREE`, `HOME` and
  `XDG_CACHE_HOME` inside the runtime.
- On Windows with `native` and a non-empty `processes:`, `lane ports` also lists
  a `pc` port. It is the supervisor control port, not an application port.
- Determine readiness from `lane status --json` (`ready`, `healthy`) instead of
  sleep loops, and read failures from `lane logs <proc>`.

## Configuration reference

A change to the runtime or named-port contract needs a new workspace: an existing
workspace keeps the contract it was created with.

| Field | Notes |
| --- | --- |
| `version` | Must be `1`. |
| `base` | Baseline branch for new workspaces; defaults to `main`. |
| `worktree_root` | Where worktrees are created. Relative values resolve inside the repository; the default is the sibling `<repo>.lanes` directory. |
| `runtime.backend` | `native` or `container`; defaults to `native`. |
| `runtime.engine` | `docker` or `podman`; container only, defaults to `docker`. |
| `runtime.image` | Prebuilt Linux image; container only and mandatory. |
| `runtime.user`, `runtime.memory`, `runtime.cpus` | Optional container overrides, e.g. `2g`, `2`. Native mode rejects them. |
| `ports` | Named ports such as `[web, pg]`. The name `pc` is reserved. |
| `listen` | Container listen port per declared port, e.g. `{web: 8080}`; every declared port needs one. |
| `env` | Extra variables for every process, hook and `lane run`. Keys may not start with `LANE_`; `GIT_DIR` and `GIT_WORK_TREE` are reserved. Values stay single-line; `${VAR}` and `$VAR` expand from the environment contract. |
| `env_file` | Repository-relative file lane writes with a managed `# BEGIN LANE`/`# END LANE` block, for host tools that read dotenv files. |
| `copy_dirs` | Repository-relative directories copied into each new workspace. |
| `hooks.setup`, `hooks.teardown` | Shell commands run in the workspace runtime; `setup` runs on create, on `up` after a failed setup, and on `reset`, and must be idempotent. |
| `processes.<name>.command` | Required for a managed process. |
| `processes.<name>.working_dir` | Defaults to the workspace root (`/workspace` in container mode). |
| `processes.<name>.environment` | A **list** of `KEY=VALUE` strings, not a mapping; lane prepends its own identity variables and refuses `LANE_*` and `GIT_*` overrides. |
| `processes.<name>.readiness_probe` | `http_get` (`host`, `port`, `path`) or `exec` (`command`), with `initial_delay_seconds`, `period_seconds`, `failure_threshold`. |
| `processes.<name>.*` | `processes` is passed through to process-compose v0.5, so its other documented fields (`log_location`, `depends_on`, `restart`, …) work too. |
| `gc.idle_stop_hours` | `lane gc` stops the runtime of a workspace unused for this long. `0` disables. |
| `gc.remove_after_days` | Age beyond which a stopped, merged workspace becomes a removal candidate. `0` disables. |
| `gc.max_workspaces` | Per-repository quota; `lane gc` evicts above it once commits are preserved. `0` disables. |

`.worktreeinclude` (repository root) lists repository-relative paths, one per
line, with `#` for comments; each is copied into a new workspace and missing
entries are skipped. It is a plain path list, not a `.gitignore` pattern file,
and it runs during setup (`lane new`, `lane reset`, `lane adopt --setup`). Use it
for local gitignored files such as `.env`; use `copy_dirs` for whole dependency
trees.

### Native: web app with a Postgres dependency
```yaml
version: 1
base: main
ports: [web, pg]
env:
  DATABASE_URL: postgres://127.0.0.1:${LANE_PORT_PG}/app
hooks:
  setup:
    - mkdir -p "$LANE_DATA_DIR/pg"
    - test -d "$LANE_DATA_DIR/pg/base" || initdb -D "$LANE_DATA_DIR/pg" --no-locale --encoding=UTF8
processes:
  pg:
    command: postgres -D "$LANE_DATA_DIR/pg" -p "$LANE_PORT_PG" -k "$LANE_DATA_DIR"
    readiness_probe:
      exec:
        command: pg_isready -h 127.0.0.1 -p "$LANE_PORT_PG"
  web:
    command: npm run dev -- --port "$LANE_PORT_WEB"
    readiness_probe:
      http_get: {host: 127.0.0.1, port: "${LANE_PORT_WEB}", path: /}
```

### Container: fixed/hardcoded listen port (e.g. 8080)
```yaml
version: 1
base: main
runtime:
  backend: container
  engine: docker
  image: lane-runtime:local
ports: [web]
listen:
  web: 8080
processes:
  web:
    command: python3 -m http.server 8080 --bind 127.0.0.1
    readiness_probe:
      http_get: {host: 127.0.0.1, port: 8080, path: /}
```

## Harness integration

`lane skill install` (also run by `lane init`) writes this skill to the single
location `.agents/skills/lane/SKILL.md`. Cursor, Codex, pi and Antigravity all
discover skills there, so there is exactly one copy to maintain. lane does not
edit `AGENTS.md`, `GEMINI.md` or any harness's own rules file.

`lane hook install [cursor|all]` merges the one lifecycle adapter that a harness
actually documents: `.cursor/worktrees.json`, which runs `lane adopt --setup`
inside every worktree Cursor creates (Agents Window, IDE or CLI), so that
worktree becomes a lane workspace with its own ports, private data and setup
hooks.

The other harnesses document no worktree-creation hook, so they are driven by
this skill instead:

- **Codex** creates its own worktrees under `$CODEX_HOME/worktrees` on a detached
  HEAD. `lane adopt` refuses a detached checkout, so those are not lane
  workspaces: use `lane new <slug>` for a task that needs ports, private data or
  services.
- **Antigravity** provisions a worktree per conversation. It exposes no
  workspace-open or worktree hook, so run `lane adopt --setup` inside it when it
  is on a branch, or `lane new <slug>` otherwise.
- **pi** has no worktree feature: every task is a `lane new <slug>`.

Session-end hooks are deliberately not installed. Codex's synchronous
`SessionEnd` is capped at a few seconds while `lane down` waits for processes to
exit, and Antigravity's `Stop`, Cursor's `sessionEnd` and pi's `session_shutdown`
do not promise to fire when a worktree or conversation goes away. Stop work
explicitly with `lane down` (keep data) or `lane done` (release the workspace),
and let `lane gc` reclaim abandoned sessions — `gc.idle_stop_hours` makes `lane
gc` stop runtimes that were left behind.

`lane adopt` needs a checked-out branch: it refuses a detached HEAD and always
refuses the primary checkout.

## Cleanup and recovery

Never use `--force` for automatic GC or to hide failed checks. `done` requires
preserved commits (merged or present upstream) and a clean worktree for
lane-owned checkouts. `--force` cannot bypass ownership, identity,
primary-worktree or shutdown checks. `adopt` is only for linked worktrees, and
`done` on an adopted checkout stops and unregisters its runtime while preserving
checkout, data and branch even with `--force`.

A failed setup leaves its worktree and state for diagnosis: fix the workspace's
configuration or hook and retry `lane new <same-name>` or `lane up`. `down`
preserves data; `reset` wipes it deliberately and only after a verified shutdown.
Unknown process or engine state means preserve the data and inspect
`lane logs <proc>` and `.lane/process-compose.log`.

`lane gc --dry-run --json` previews collection. GC skips active operations and
unpreserved commits. `doctor --fix` installs native dependencies such as
process-compose; it never performs destructive GC.
