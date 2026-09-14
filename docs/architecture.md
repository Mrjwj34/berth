# Architecture decision: explicit runtimes, safe lifecycle

Goal: low-overhead local environments for parallel agent development.

## Boundaries

- `cmd/berth`: CLI parsing, JSON/text output, cancellation and exit status.
- `app`: ownership, operation locking, setup state, safe lifecycle and configuration scope.
- `runner.Session`: the single execution context for services, run, hooks and probes.
- `process`: native process-compose adapter, rendering, readiness and pinned binary installation.
- `worktree`, `copyfs`, `ports`, `state`: narrow reusable primitives.
- `gc`: conservative eligibility selection, refreshed under the lifecycle lock.

There is no libc/dyld/seccomp injection backend. Adding a third OS-specific syscall
compatibility implementation is not necessary for the core parallel-development
workflow. The native fast path remains independent of containers. The isolated
path reuses a shared Linux engine and a prepared image, with one namespace/container
per workspace. Future engine-specific optimizations belong behind Session rather
than in individual command handlers.

A separate daemon is deliberately not introduced: registry updates are short,
workspace operations hold only their own lock, and the existing process-compose
or container engine supplies long-lived supervision. Separate workspace operations
can proceed concurrently. Commands in the same workspace are serialized; a shared
read/execution lease is a possible later extension if measurements justify it.

## Invariants

1. Primary repositories, their ancestors and wrong Git identities are never removal targets.
2. Automatic GC never grants force authority. Clean is not equivalent to preserved.
3. Imported/adopted records never gain deletion ownership through migration.
4. Port allocation and registry insertion form one cross-process transaction.
5. Unknown process/engine state prevents destructive cleanup.
6. Incomplete setup is recorded and retried, not treated as completed initialization.
7. Copies of writable dependencies are independent, including fallback paths.
8. Service, test, migration, hook and probe share the workspace execution context.
9. Image/backend failure does not silently run a command on the host instead.
10. JSON stdout is not mixed with setup/teardown command output.

The registry moves from version 1 to 2. Legacy ownership stays empty until explicit
adoption, which grants only adopted ownership. A crash after Git creates a checkout
but before berth registers it leaves an unregistered checkout that can be adopted;
no destructive guess is made. Runtime control endpoints live outside newly created
native checkouts. PID checks after native shutdown are only used to wait, never to
kill a potentially reused PID. Native API tokens isolate control endpoints.
Graceful shutdown is bounded by an injected process-compose `shutdown.timeout_seconds`
on Unix native and container runtimes, so the supervisor escalates to SIGKILL
without berth killing a PID it did not spawn. Native Windows keeps the supervisor
default because process-compose already terminates with `taskkill /T /F`.

Recovery intent is stored in the existing workspace record, under its operation
lock: `reset_pending` precedes runtime/data destruction; `removal_head` and
`removal_branch` follow teardown/shutdown and precede checkout removal. A workspace
is identified by its path and Git directory, not by branch name, so a worktree that
moves to `fix/*` stays usable. Reset retries finish the reset before setup. Removal
retries retain repository identity checks, delete only the recorded commit using
[Git's compare-and-delete](https://git-scm.com/docs/git-update-ref), and delete a
branch only while it is still the original `berth/<slug>`.
GC never discards pending removal records or grants them force authority. No
separate journal service or background recovery process is needed.

## Validation

Unit/regression coverage includes independent copies, same-source copies,
primary-checkout protection, adopted checkout preservation, clean unpublished
commits, setup retry using branch-local config, unknown-stop safety, GC versus
active operations, independent-process allocation, runtime argument generation,
host/internal port semantics, strict config, checksums and readiness. Recovery
tests interrupt reset with invalid data paths and removal with real Git ref locks,
including changed, already-deleted and checked-out branch cases.

The Linux container acceptance test holds a host sentinel on the same original
port used inside two concurrent workspaces. It verifies the host sentinel and
workspace servers do not get confused, and checks hooks, private data, Git,
argv/shell execution, reuse across up/down, cancellation, sibling survival and
safe removal. Missing engines/images fail this test rather than being skipped.

CI runs native CLI acceptance on all three hosted OSes: parallel workspace
creation, real HTTP, JSON, hooks, argv/exit codes, restart, reset and safe removal.
Docker/Podman Desktop execution on macOS/Windows and rootless engine variations
still require those actual environments. Benchmarks should separate cold image
preparation, warm workspace creation, dependency copying, process readiness and
incremental run latency. No unmeasured latency or memory figure is a release SLO.
