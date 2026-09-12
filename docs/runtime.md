# Runtime contract

`native` is the low-overhead fast path for configurable applications. It provides
worktree lifecycle, port reservations, per-worktree data and native process
supervision. It does not isolate hardcoded loopback ports, absolute storage paths,
Unix sockets, registries, shared credentials or arbitrary host access.

`container` supplies one reusable Linux container per workspace using an existing
local Docker or Podman engine. No VM or image is created per command. A prepared
image must provide bash, sh, sleep, socat, git, process-compose v1.122.0 and the
project's tools. The checked-in Dockerfile is a minimal example, not an attempt to
bundle every language. Mutable image tags can change after reset/recreation; use
an image digest where reproducibility matters.

## Execution and addresses

The whole workspace's service graph, hooks, probes and one-off commands execute
in one runtime. Container cwd is `/workspace`, data is `/workspace/.berth/data`.
A writable bind mount carries the checkout; shared Git metadata is mounted at
`/berth/git`. GIT_DIR/GIT_WORK_TREE point Git at the corresponding linked-worktree
metadata. The managed HOME is private to the container. Host credentials and the
Docker socket are not mounted automatically.

The process definitions retain process-compose syntax. Native hooks use sh on
Unix and cmd on Windows; arbitrary shell strings are not made portable. Container
hooks use sh. `run` preserves argv, while `run -- sh -c ...` intentionally invokes
a shell. Rootless/user mapping and mounted filesystem permissions depend on the
engine; runtime.user can override the default. Native Linux uses the host UID/GID
inside Docker; rootless Podman uses keep-id. Rootless Docker may require an explicit
runtime.user override matching its user namespace.

`up` waits for declared health probes to report ready. A running process without
a probe is accepted as started; this does not guarantee its application endpoint
is ready. Declare a readiness probe when dependent commands need that guarantee.
Allow for cold startup in the probe's initial delay and failure threshold:
process-compose stops a process when its readiness failure threshold is reached
(three failures by default). Native `up` has a 60-second readiness deadline;
raising the probe threshold does not remove that deadline or bypass readiness.

`ports` names host publications. `listen` specifies their original TCP port in
container mode. Every named port requires a listen value. Internal services can
also use undeclared ports, but berth does not dynamically discover/publish them.
`BERTH_PORT_*` means the current execution context; `BERTH_HOST_PORT_*` means the
host publication. Browser code running outside the container must use published
URLs rather than blindly reusing internal localhost URLs.

Publications bind to host 127.0.0.1. An internal socat gateway forwards from a high
container port to the application's IPv4 loopback listener. `plan.gateway_ports`
lists reserved gateway ports; applications must not bind those gateway ports.
Declaring the application's listeners lets the allocator avoid them, including
listeners in 20000..39999. Gateways are supervised by the container entrypoint;
a gateway's exit stops the container rather than silently disabling a publication.
The publication contract is TCP/IPv4, not UDP or IPv6-only loopback. Use a current
engine; old Docker versions have different localhost-publishing security behavior.

Separate network namespaces prevent internal loopback/port collisions. They are
not egress firewalls and do not forbid reaching other host publications. Network
access uses the engine's normal bridge/NAT/DNS configuration. Unix sockets stored
at workspace-relative locations remain distinct; shared host paths do not.

## Lifetime and recovery

A stored runtime contract (backend, image reference, engine, limits, named ports)
is immutable for an existing workspace. Changing processes/hooks/env in the
workspace is supported. Changing the runtime/port contract requires a new
workspace. This avoids silently changing meaning or controlling unrelated
resources while partial state is present.

Container name and labels must match the stored workspace ID, canonical path and
runtime specification before stop/remove. Daemon failure is not treated as
container absence. No implicit pull, build, or native fallback occurs. Setup is
idempotent by project responsibility: berth records completion, and retries an
incomplete setup. Projects must not assume arbitrary hook effects can be rolled
back transactionally.

`down` stops the workspace, preserving its checkout/data and container instance.
`reset` destroys the stopped runtime, resets private data, recreates it and reruns
setup. `done` never removes an adopted checkout. Any stop/identity failure blocks
destructive work. Cancellation of `run` in container mode stops that workspace's
whole container; sibling services within that workspace stop too, but other
workspaces remain running.

Reset intent is persisted before changing the runtime or data. If reset fails or
is interrupted, the next `new`, `up`, or `run` finishes the data reset before
retrying setup. Fix the reported filesystem/engine error first; partial data is
not treated as an initialized environment.

After teardown and shutdown, `done` records the expected branch commit before
removing the checkout. Retrying `done` can finish branch deletion and registration
cleanup even after the checkout is gone. A replaced checkout, remaining Git
worktree metadata, a checked-out branch or changed branch commit blocks recovery;
`--force` does not override those checks. Branch deletion uses Git's atomic
old-value check. GC retains these pending records and asks for an explicit `done`
retry. Adopted workspaces retain their metadata as well as their data and branch.

The runtime and repository data are not an adversarial security boundary: shared
Git metadata is writable, user hooks are code, and explicitly supplied config may
contain side effects. An untrusted-agent sandbox needs separate credentials,
filesystem policy, egress policy and resource enforcement. Do not describe this
runner as a complete security sandbox.

## Windows

A native process on Windows cannot receive an argument that contains a space,
and it cannot receive a quoted argument. The cause is in the chain, not in the
configuration: process-compose starts the command as `["cmd", "/C", <the whole
command string>]`, the Go runtime escapes the quote characters inside that
string as `\"`, and `cmd.exe` treats a backslash as an ordinary character. The
quote therefore arrives as data and the space behind it stops grouping words.
Observed in `process-compose.log` for a command that quotes its data directory:

```
INF Started command=["cmd","/C","go run ./cmd/server --db \"d:\\...\\smoke-a\\.berth\\data/rei.db\""]
[server] exit status 1
```

Three fixes work, in order of preference:

1. **Drop the quotes and keep every argument space-free.** Ports always qualify;
   `--listen 127.0.0.1:${BERTH_PORT_SERVER}` is fine unquoted.
2. **Give the workspace a space-free path** so absolute paths do too: set
   `worktree_root` to a directory without spaces, for example
   `worktree_root: C:/berth-workspaces`. `BERTH_DATA_DIR` is then space-free and
   `--db ${BERTH_DATA_DIR}/rei.db` works unquoted. A relative path such as
   `.berth/data/rei.db` works as well, because berth sets `working_dir` to the
   workspace root.
3. **Read the value from the environment inside the program.** The environment is
   passed as a block and is never split, so `BERTH_DATA_DIR` keeps its spaces.

Use `${BERTH_DATA_DIR}`, not `%BERTH_DATA_DIR%`: berth substitutes the
`${...}` form itself, and the `%...%` form is expanded later by `cmd`, if at all.
Quoting is still correct on Linux and macOS, where commands run through a shell.

berth warns on `up` when a declared command cannot work this way, naming the
process and the fix. Container workspaces are unaffected, because their commands
run inside Linux. The supervisor itself also needs `BERTH_HOME` on a path without
spaces on Windows.

### When the supervisor state is unknown

If the control files exist but the supervisor does not answer, berth treats the
process state as unknown and refuses every operation that could destroy data,
including `done --force`. That is deliberate: the alternative is removing a
checkout while processes still hold it.

The control files are not inside the workspace. On Windows they live under
`BERTH_HOME/run/<hash>/` as `pc.port` and `pc.token`; on Unix the socket is
`<workspace>/.berth/pc.sock` with a `pc.token` beside it. When no
`process-compose` is running for that workspace, remove the stale pair, then
retry `down` and `done`. `pc.yaml`, `data` and the checkout are untouched by that
cleanup.
