# Features and Configuration Guide

berth is a developer tool designed to manage parallel development environments on a local machine. It allows multiple software agents or human developers to work on separate tasks concurrently without port collisions, state corruption, or heavy virtual machine overhead.

This guide provides a comprehensive walkthrough of berth concepts, configuration syntax, practical examples, and commands.

## Core Concepts

### Workspaces

A workspace in berth combines an isolated Git worktree, a private data directory, dynamically allocated ports, and a set of supervised background processes. Each workspace checked out from Git uses its own branch configuration. Changes made to configuration files in one workspace do not affect other workspaces.

Workspaces are created beside the repository in `<repo>.berths/<slug>` (or under `worktree_root`) on branch `berth/<slug>`, and a slug is lowercase ASCII with digits, `-` or `_` only. The primary checkout is never a workspace, and the commands that take no slug (`berth run`, `berth open`) resolve the workspace from the current directory, so run them from inside the path `berth new` printed.

### Dual Runtime Modes

berth supports two distinct execution backends:

Native mode runs ordinary processes directly on the host operating system. It relies on environment variables and command line flags to assign dynamic ports and data paths. This mode offers maximum performance and zero container overhead.

Container mode runs a reusable Linux container per workspace using an existing local Docker or Podman installation. It preserves hardcoded loopback ports through internal forwarding gateways. If an application requires binding to port 8080 or port 5432, container mode isolates the network namespace while forwarding host traffic cleanly.

### Port Allocation

When a workspace declares named ports in its configuration, berth assigns unique host ports using an atomic reservation transaction. These assignments remain stable throughout the lifetime of the workspace. Programs access allocated ports through environment variables such as BERTH_PORT_WEB for internal services and BERTH_HOST_PORT_WEB for external host access.

### Private Data and State Isolation

Each workspace receives a dedicated data directory located at .berth/data in native mode and at /workspace/.berth/data in container mode. Databases, cache stores, and scratch files remain isolated to the active workspace. Writable dependencies can be copied using Copy on Write on supported filesystems such as Linux reflink and macOS clonefile to avoid physical disk duplication.

### Daemonless Operation

berth does not run a continuous background daemon. Instead, it relies on file locks and a single machine-level registry file. Long operations hold individual workspace locks rather than a global machine lock. Process supervision is delegated to process-compose or the container runtime.

## Configuration Reference: berth.yaml

Every workspace reads its configuration from berth.yaml located at the root of the repository.

### Top-Level Fields

- version: Configuration format version. Currently set to 1.
- base: Default baseline branch for new worktrees, such as main.
- worktree_root: Optional directory path where new worktrees are created. Defaults to sibling directory named dot-berths.
- runtime: Runtime execution settings.
  - backend: native or container. Defaults to native.
  - engine: docker or podman when container backend is selected.
  - image: Name of the prebuilt Linux container image.
  - memory: Optional memory limit such as 2g.
  - cpus: Optional CPU limit such as 2.
  - user: Optional UID and GID override for container processes.
- ports: List of named ports required by the project, such as web or pg.
- listen: Mapping of named ports to internal TCP listen ports when using container mode.
- env: Key-value map of environment variables injected into processes and hooks.
- copy_dirs: List of repository directories copied into each new worktree using Copy on Write where available.
- hooks: Lifecycle shell commands.
  - setup: List of commands executed when a workspace is created or reset.
  - teardown: List of commands executed before an owned workspace is removed.
- processes: Mapping of background services managed by berth.
  - command: Command string to execute.
  - working_dir: Working directory for the process. Defaults to the workspace root.
  - environment: Additional environment variables for this process, given as a list of KEY=VALUE strings, not as a mapping. berth prepends its identity variables and rejects BERTH_* and GIT_* overrides.
  - readiness_probe: Probe used to verify service availability during startup.
    - http_get: HTTP readiness probe specifying host, port, and path.
    - exec: Command readiness probe executing a shell command.
    - initial_delay_seconds: Seconds to wait before first probe attempt.
    - period_seconds: Interval between probe attempts.
    - failure_threshold: Consecutive failures before marking the service unready.
  - log_location: Optional per-process log file, as used by the acceptance test. The processes mapping is passed through to process-compose v0.5, so its other documented fields such as depends_on and restart also work.

### Worktree Include File

A `.worktreeinclude` file at the repository root lists repository-relative paths, one per line, with `#` for comments. Each listed path is copied into a new worktree during setup (`berth new`, `berth reset`, `berth adopt --setup`), and entries that do not exist are skipped. It is a plain path list, not a `.gitignore` pattern file: use it for local gitignored files such as `.env`, and `copy_dirs` for whole dependency trees.

## Practical Configuration Examples

### Example 1: Native Node.js Web Application

```yaml
version: 1
base: main
ports: [web]
env:
  PORT: ${BERTH_PORT_WEB}
processes:
  web:
    command: npm run dev -- --port "$BERTH_PORT_WEB"
    readiness_probe:
      http_get:
        host: 127.0.0.1
        port: "${BERTH_PORT_WEB}"
        path: /
```

### Example 2: Native App with Managed PostgreSQL Service

```yaml
version: 1
base: main
ports: [web, pg]
env:
  PORT: ${BERTH_PORT_WEB}
  DATABASE_URL: postgres://127.0.0.1:${BERTH_PORT_PG}/app
hooks:
  setup:
    - mkdir -p "$BERTH_DATA_DIR/pg"
    - test -d "$BERTH_DATA_DIR/pg/base" || initdb -D "$BERTH_DATA_DIR/pg" --no-locale --encoding=UTF8
processes:
  pg:
    command: >
      postgres -D "$BERTH_DATA_DIR/pg" -p "$BERTH_PORT_PG" -k "$BERTH_DATA_DIR"
      -c fsync=off -c synchronous_commit=off
    readiness_probe:
      exec:
        command: pg_isready -h 127.0.0.1 -p "$BERTH_PORT_PG"
  web:
    command: npm run dev -- --port "$BERTH_PORT_WEB"
    readiness_probe:
      http_get:
        host: 127.0.0.1
        port: "${BERTH_PORT_WEB}"
        path: /
```

### Example 3: Container Mode with Fixed Listen Port

When your application hardcodes listening on port 8080 and cannot read dynamic environment variables:

```yaml
version: 1
base: main
runtime:
  backend: container
  engine: docker
  image: berth-runtime:local
ports: [web]
listen:
  web: 8080
processes:
  web:
    command: python3 -m http.server 8080 --bind 127.0.0.1
    readiness_probe:
      http_get:
        host: 127.0.0.1
        port: 8080
        path: /
```

### Example 4: Accelerating Data Initialization with Copy on Write

For large databases, running full database migrations in every workspace can be slow. You can maintain a clean initialized template directory in your repository or host filesystem and clone it instantaneously:

```yaml
version: 1
base: main
ports: [pg]
hooks:
  setup:
    - |
      if [ ! -d "$BERTH_DATA_DIR/pg/base" ]; then
        cp -a --reflink=auto .seed/pg "$BERTH_DATA_DIR/pg" 2>/dev/null || \
        (mkdir -p "$BERTH_DATA_DIR/pg" && initdb -D "$BERTH_DATA_DIR/pg" --no-locale --encoding=UTF8)
      fi
```

On Linux with btrfs or xfs, and macOS with APFS, files clone in milliseconds without consuming initial disk space.

## Environment Variables Reference

berth injects the following variables into every hook, service, and berth run command:

- BERTH_WORKSPACE: Absolute path to the workspace directory.
- BERTH_ROOT: Path to the primary Git repository.
- BERTH_DATA_DIR: Private data storage path for this workspace.
- BERTH_SLUG: Workspace slug identifier.
- BERTH_BRANCH: Git branch name associated with this workspace.
- BERTH_PORT_NAME: Internal listening port for the named service.
- BERTH_HOST_PORT_NAME: Published host port on 127.0.0.1 for browser access.

In container mode, additional Git metadata variables are provided:
- GIT_DIR: Internal linked worktree metadata path.
- GIT_WORK_TREE: Mounted workspace root at slash-workspace.
- HOME: Private container home directory.
- XDG_CACHE_HOME: Private container cache path.

## Command Reference

| Command | Purpose | Key Flags |
| --- | --- | --- |
| berth init | Create berth.yaml template and install agent skills | --force |
| berth new slug | Create an isolated workspace and worktree | --up, --base branch |
| berth adopt | Register an existing external Git worktree | --setup |
| berth attach slug | Print workspace paths and shell export statements | --json |
| berth ls | List all active workspaces and their status | --json |
| berth status slug | Inspect process statuses and readiness probes | --json |
| berth ports slug | View allocated ports and listen mappings | --json |
| berth plan slug | Review the runtime contract before execution | --json |
| berth up slug | Start all declared workspace background processes | |
| berth down slug | Gracefully stop running processes without deleting data | |
| berth logs proc slug | Stream stdout and stderr logs for a service | |
| berth run -- cmd | Execute a command within the workspace environment | |
| berth reset slug | Wipe private data directory and rerun setup hooks | |
| berth done slug | Verify commit preservation and safely remove workspace | --force |
| berth gc | Clean up orphaned registrations and inactive workspaces | --dry-run, --json |
| berth doctor | Validate system dependencies and state health | --fix, --json |
| berth open port slug | Open service publication URL in the host browser | --json |
| berth skill install | Install the embedded SKILL.md into `.agents/skills/berth` | |
| berth hook install | Merge the Cursor worktree adapter into `.cursor/worktrees.json` | cursor, all |

## Integration with Coding Agents

berth is designed specifically for autonomous programming agents.

### Installing Skills and Hooks

berth keeps one skill and at most one adapter per harness, so a repository never grows a copy for every tool.

`berth init` and `berth skill install` deploy the skill to the single location `.agents/skills/berth/SKILL.md`. Cursor, Codex, pi and Antigravity all discover skills in `.agents/skills`, so there is exactly one copy to keep up to date. berth writes nothing into `AGENTS.md`, `GEMINI.md` or any harness's own rules file.

`berth hook install [cursor|all]` merges `.cursor/worktrees.json` with `berth adopt --setup`, which runs inside every worktree Cursor creates in the Agents Window, the IDE or the CLI. That is the only worktree-creation hook a supported harness documents: Codex creates detached worktrees under `$CODEX_HOME/worktrees`, Antigravity provisions a worktree per conversation, and pi has no worktree feature at all, so those harnesses are driven by the skill instead. Session-end hooks are deliberately not installed, because Codex caps its synchronous `SessionEnd` at a few seconds while `berth down` waits for processes to exit, and the other harnesses do not promise an event when a worktree or conversation goes away. Stop work explicitly with `berth down` (keep data) or `berth done` (release the workspace), and let `berth gc` reclaim what was abandoned — `gc.idle_stop_hours` makes it stop idle runtimes.

### Guiding Your Agent

When working with an agent, you can ask it to perform tasks directly in isolated workspaces:

> Create a new workspace named auth-refactor using berth, start the services, and implement the token renewal endpoint.

The agent uses the embedded skill to interact with berth commands, verify service health, and run tests within the dedicated environment.
