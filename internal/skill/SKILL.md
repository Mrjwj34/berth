---
name: lane
description: Manage independent local agent workspaces, explicit native/container runtimes, processes and safe cleanup.
---

# lane

Use lane for parallel local development, independent Git worktrees, private data
and managed service graphs. lane is not an adversarial security sandbox. Do not
replace lane lifecycle operations with raw rm -rf or guessed PID/port cleanup.

## Onboard

Read the project's startup scripts, toolchains, storage paths and listen ports.
Choose native for already configurable programs. Choose container for unchanged
hardcoded ports or a uniform Linux environment. Do not attempt preload/seccomp
injection. The Linux engine and prepared image must exist; failures must not be
worked around by silently running the command on the host.

Commit lane.yaml before creating a workspace. New workspaces execute their own
branch-local configuration. Native `ports: [web]` allocates LANE_PORT_WEB; use
existing command/config options. Container mode requires runtime.backend:
container, an existing runtime.image, named ports and listen mappings. Include
project tools, bash, sh, sleep, socat, Git and process-compose v1.122.0 in the image.
Build/install dependencies once per image rather than every run where possible.

Use workspace-private data. Copying mutable dependencies uses CoW/plain copy, not
hardlinks. External symlinks and absolute shared storage need explicit treatment.

## Configuration Examples

### Native: Web App with Postgres Dependency
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

### Container: Fixed/Hardcoded Listen Port (e.g. 8080)
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

## Execute

```sh
lane new task-name --up --json
lane plan task-name
lane status --json
lane ports --json
lane run -- <program> <args...>
lane logs <process>
```

Use lane run for tests/migrations/one-off work: it joins the same runtime as the
services and hooks. Arguments are not reparsed. Request a shell explicitly with
`lane run -- sh -c '...'` in Linux/container mode, or the appropriate native shell.
Container cancellation stops that workspace's runtime, including its sibling
services. Do not assume a shell script is portable between sh and Windows cmd.

LANE_PORT_* is the port in the execution context. LANE_HOST_PORT_* is the host
publication. A browser outside the runtime must use the host port. Container
publications are TCP over IPv4 loopback. `plan.gateway_ports` reserves internal
forwarder ports; application listeners must not use those values. Undeclared
internal ports are isolated but not automatically discovered/published.

## Cleanup and recovery

Never use --force for automatic GC or to hide failed checks. `done` requires
preserved commits and a clean worktree for lane-owned checkouts. `--force` cannot
bypass ownership, primary-worktree protection, identity or shutdown checks.
`adopt` is only for linked worktrees. `done` on an adopted checkout stops and
unregisters its runtime, preserving checkout/data/branch even with --force.

A failed setup leaves its worktree and state for diagnosis. Fix the workspace's
config/hook and retry `lane new <same-name>` or `lane up`; hooks must be idempotent.
A runtime or named-port contract change needs a new workspace. Use `down` to
preserve data, `reset` to intentionally reset it only after verified shutdown.

`lane gc --dry-run --json` previews collection. GC skips active operations and
unpreserved commits. `doctor --fix` installs native dependencies; it never performs
destructive GC. Unknown process/engine state means preserve data and inspect logs.
