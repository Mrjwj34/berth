# Features and Configuration Guide

lane is a developer tool designed to manage parallel development environments on a local machine. It allows multiple software agents or human developers to work on separate tasks concurrently without port collisions, state corruption, or heavy virtual machine overhead.

This guide provides a comprehensive walkthrough of lane concepts, configuration syntax, practical examples, and commands.

## Core Concepts

### Workspaces

A workspace in lane combines an isolated Git worktree, a private data directory, dynamically allocated ports, and a set of supervised background processes. Each workspace checked out from Git uses its own branch configuration. Changes made to configuration files in one workspace do not affect other workspaces.

Workspaces are created beside the repository in `<repo>.lanes/<slug>` (or under `worktree_root`) on branch `lane/<slug>`, and a slug is lowercase ASCII with digits, `-` or `_` only. The primary checkout is never a workspace, and the commands that take no slug (`lane run`, `lane open`) resolve the workspace from the current directory, so run them from inside the path `lane new` printed.

### Dual Runtime Modes

lane supports two distinct execution backends:

Native mode runs ordinary processes directly on the host operating system. It relies on environment variables and command line flags to assign dynamic ports and data paths. This mode offers maximum performance and zero container overhead.

Container mode runs a reusable Linux container per workspace using an existing local Docker or Podman installation. It preserves hardcoded loopback ports through internal forwarding gateways. If an application requires binding to port 8080 or port 5432, container mode isolates the network namespace while forwarding host traffic cleanly.

### Port Allocation

When a workspace declares named ports in its configuration, lane assigns unique host ports using an atomic reservation transaction. These assignments remain stable throughout the lifetime of the workspace. Programs access allocated ports through environment variables such as LANE_PORT_WEB for internal services and LANE_HOST_PORT_WEB for external host access.

### Private Data and State Isolation

Each workspace receives a dedicated data directory located at .lane/data in native mode and at /workspace/.lane/data in container mode. Databases, cache stores, and scratch files remain isolated to the active workspace. Writable dependencies can be copied using Copy on Write on supported filesystems such as Linux reflink and macOS clonefile to avoid physical disk duplication.

### Daemonless Operation

lane does not run a continuous background daemon. Instead, it relies on file locks and a single machine-level registry file. Long operations hold individual workspace locks rather than a global machine lock. Process supervision is delegated to process-compose or the container runtime.

## Configuration Reference: lane.yaml

Every workspace reads its configuration from lane.yaml located at the root of the repository.

### Top-Level Fields

- version: Configuration format version. Currently set to 1.
- base: Default baseline branch for new worktrees, such as main.
- worktree_root: Optional directory path where new worktrees are created. Defaults to sibling directory named dot-lanes.
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
- processes: Mapping of background services managed by lane.
  - command: Command string to execute.
  - working_dir: Working directory for the process. Defaults to the workspace root.
  - environment: Additional environment variables for this process, given as a list of KEY=VALUE strings, not as a mapping. lane prepends its identity variables and rejects LANE_* and GIT_* overrides.
  - readiness_probe: Probe used to verify service availability during startup.
    - http_get: HTTP readiness probe specifying host, port, and path.
    - exec: Command readiness probe executing a shell command.
    - initial_delay_seconds: Seconds to wait before first probe attempt.
    - period_seconds: Interval between probe attempts.
    - failure_threshold: Consecutive failures before marking the service unready.
  - log_location: Optional per-process log file, as in the acceptance test. The processes mapping is passed through to process-compose v0.5, so its other documented fields such as depends_on and restart also work.

### Worktree Include File

A `.worktreeinclude` file at the repository root lists repository-relative paths,
one per line, with `#` for comments. Each listed path is copied into a new
worktree during setup (`lane new`, `lane reset`, `lane adopt --setup`), and
entries that do not exist are skipped. It is a plain path list, not a
`.gitignore` pattern file: use it for local gitignored files such as `.env`, and
`copy_dirs` for whole dependency trees.

## Practical Configuration Examples

### Example 1: Native Node.js Web Application

```yaml
version: 1
base: main
ports: [web]
env:
  PORT: ${LANE_PORT_WEB}
processes:
  web:
    command: npm run dev -- --port "$LANE_PORT_WEB"
    readiness_probe:
      http_get:
        host: 127.0.0.1
        port: "${LANE_PORT_WEB}"
        path: /
```

### Example 2: Native App with Managed PostgreSQL Service

```yaml
version: 1
base: main
ports: [web, pg]
env:
  PORT: ${LANE_PORT_WEB}
  DATABASE_URL: postgres://127.0.0.1:${LANE_PORT_PG}/app
hooks:
  setup:
    - mkdir -p "$LANE_DATA_DIR/pg"
    - test -d "$LANE_DATA_DIR/pg/base" || initdb -D "$LANE_DATA_DIR/pg" --no-locale --encoding=UTF8
processes:
  pg:
    command: >
      postgres -D "$LANE_DATA_DIR/pg" -p "$LANE_PORT_PG" -k "$LANE_DATA_DIR"
      -c fsync=off -c synchronous_commit=off
    readiness_probe:
      exec:
        command: pg_isready -h 127.0.0.1 -p "$LANE_PORT_PG"
  web:
    command: npm run dev -- --port "$LANE_PORT_WEB"
    readiness_probe:
      http_get:
        host: 127.0.0.1
        port: "${LANE_PORT_WEB}"
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
  image: lane-runtime:local
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
      if [ ! -d "$LANE_DATA_DIR/pg/base" ]; then
        cp -a --reflink=auto .seed/pg "$LANE_DATA_DIR/pg" 2>/dev/null || \
        (mkdir -p "$LANE_DATA_DIR/pg" && initdb -D "$LANE_DATA_DIR/pg" --no-locale --encoding=UTF8)
      fi
```

On Linux with btrfs or xfs, and macOS with APFS, files clone in milliseconds without consuming initial disk space.

## Environment Variables Reference

lane injects the following variables into every hook, service, and lane run command:

- LANE_WORKSPACE: Absolute path to the workspace directory.
- LANE_ROOT: Path to the primary Git repository.
- LANE_DATA_DIR: Private data storage path for this workspace.
- LANE_SLUG: Workspace slug identifier.
- LANE_BRANCH: Git branch name associated with this workspace.
- LANE_PORT_NAME: Internal listening port for the named service.
- LANE_HOST_PORT_NAME: Published host port on 127.0.0.1 for browser access.

In container mode, additional Git metadata variables are provided:
- GIT_DIR: Internal linked worktree metadata path.
- GIT_WORK_TREE: Mounted workspace root at slash-workspace.
- HOME: Private container home directory.
- XDG_CACHE_HOME: Private container cache path.

## Command Reference

| Command | Purpose | Key Flags |
| --- | --- | --- |
| lane init | Create lane.yaml template and install agent skills | --force |
| lane new slug | Create an isolated workspace and worktree | --up, --base branch |
| lane adopt | Register an existing external Git worktree | --setup |
| lane attach slug | Print workspace paths and shell export statements | --json |
| lane ls | List all active workspaces and their status | --json |
| lane status slug | Inspect process statuses and readiness probes | --json |
| lane ports slug | View allocated ports and listen mappings | --json |
| lane plan slug | Review the runtime contract before execution | --json |
| lane up slug | Start all declared workspace background processes | |
| lane down slug | Gracefully stop running processes without deleting data | |
| lane logs proc slug | Stream stdout and stderr logs for a service | |
| lane run -- cmd | Execute a command within the workspace environment | |
| lane reset slug | Wipe private data directory and rerun setup hooks | |
| lane done slug | Verify commit preservation and safely remove workspace | --force |
| lane gc | Clean up orphaned registrations and inactive workspaces | --dry-run, --json |
| lane doctor | Validate system dependencies and state health | --fix, --json |
| lane open port slug | Open service publication URL in the host browser | --json |
| lane skill install | Install the embedded SKILL.md into `.agents/skills/lane` | |
| lane hook install | Install the `.agents/hooks` scripts and the harness adapter files | cursor, claude, scripts, all |

## Integration with Coding Agents

lane is designed specifically for autonomous programming agents.

### Installing Skills and Hooks

lane keeps its agent assets in one directory per asset kind, so a repository does
not grow a copy for every harness.

`lane init` and `lane skill install` deploy the skill to the single location
`.agents/skills/lane/SKILL.md`. Harnesses that read `.agents/skills` (Cursor,
DSH, Codex-style agents) discover it directly. Claude Code reads
`.claude/skills/<name>/`, so give it one symlink rather than a second copy:

```sh
ln -s ../../.agents/skills/lane .claude/skills/lane
```

`lane hook install all` writes the shared hook scripts to `.agents/hooks/` and
then the adapter file of each requested harness:

- `.claude/settings.json` merges `WorktreeCreate` and `WorktreeRemove` handlers
  that run the `.agents/hooks/claude-worktree-*.sh` scripts through `bash`, so
  they do not depend on an executable bit surviving a checkout.
- `.cursor/worktrees.json` merges `lane adopt --setup`, which registers Cursor's
  own worktrees with lane.

Use `lane hook install scripts` to write the shared scripts without any adapter.
Hook behavior and limits: creation runs `lane new <name> --print-path`, which
applies setup but does not start services; removal runs `lane down` only, leaving
the registration for `lane gc` to reclaim. The Claude scripts need `bash`,
`python3` and `lane` on `PATH` and are Unix-only, so on Windows install Git Bash
or run `lane new` / `lane adopt --setup` directly. Adapter files belong to the
harness configuration and are meant to be committed, so re-run
`lane hook install` in a fresh clone instead of assuming the hooks are active.

### Guiding Your Agent

When working with an agent, you can ask it to perform tasks directly in isolated workspaces:

> Create a new workspace named auth-refactor using lane, start the services, and implement the token renewal endpoint.

The agent uses the embedded skill to interact with lane commands, verify service health, and run tests within the dedicated environment.
