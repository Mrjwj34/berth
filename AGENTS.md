# Repository development rules

berth provides efficient, low-cost local environments for parallel agent
programming. Preserve the native fast path and explicit reusable Linux-container
backend. Do not reintroduce transparent syscall interception as a universal
isolation promise. No per-command image builds, implicit image pulls or silent
fallback from isolated to host execution.

## Contribution flow

`main` is PR-only. Never commit or push to `main`: branch, push the branch and
open a pull request (`gh pr create --fill`), then stop — a pull request is merged
only when the user asks for it. The `main` ruleset rejects direct pushes, force
pushes and deletion for every actor, with no bypass, so a direct push fails at
the remote; `git config core.hooksPath .githooks` installs the committed
pre-commit guard that refuses the commit locally instead. The pull request title
carries the conventional commit type, and the pull request body follows
`.github/PULL_REQUEST_TEMPLATE.md`.

## Architecture

`app` owns lifecycle, ownership and locks. `runner.Session` is the one execution
context shared by up/run/hooks/probes. `process` adapts process-compose;
`worktree`, `copyfs`, `ports`, `state` are narrow primitives. `gc` selects safe
candidates and revalidates them under the same workspace lock. Avoid a new daemon
or generic plugin framework without a measured need.

Project components are ordinary commands, not built-in database types. Runtime
and named-port contracts are immutable for an existing workspace. Branch-local
process/env/hook configuration is authoritative. Host/internal addresses must be
explicit and observable through JSON/plan.

## Safety and correctness

- Never recursively remove a checkout after Git refuses worktree removal.
- --force cannot bypass ownership, identity, primary-worktree or shutdown checks.
- Adopted/legacy worktrees never receive implicit deletion authority.
- GC never uses force; clean is not equivalent to preserved/pushed/merged.
- Stop errors and unknown runtime state block destructive operations.
- Port reservation plus workspace insertion is one cross-process transaction.
- Long operations hold a workspace lock, not the machine registry lock.
- Writable files use CoW or independent copies, never hardlink fallback.
- Setup completion is persisted; failed setup must be safely retryable.
- Hooks use the workspace runtime and stderr, preserving CLI JSON stdout.
- Preserve argv; do not prepend strings to arbitrary shell programs.
- Container runtime is not a hostile-code sandbox: Git metadata is shared.

## Validation

Go version is defined by go.mod. Before opening a pull request:

```sh
test -z "$(gofmt -l cmd internal)"
go vet ./cmd/... ./internal/...
go test ./cmd/... ./internal/...
go test -race ./cmd/... ./internal/...
go run honnef.co/go/tools/cmd/staticcheck@v0.7.0 ./cmd/... ./internal/...   # the toolchain go.mod defines
go build ./cmd/berth
```

Run these with the toolchain go.mod defines. The pinned staticcheck decodes the
export data of the compiler that built the packages, so a newer local toolchain
fails it with an export-data version error, and a newer staticcheck refuses to
build on an older one: moving the pin means moving go.mod with it.

Filesystem/Git tests use t.TempDir and isolated BERTH_HOME. Tests involving registry
allocation must cover independent CLI processes, not just go test -race. Claimed
platform capabilities require execution tests; skipped tests and cross-compilation
are not equivalent to native/desktop-runtime validation. Run the real container
acceptance test with a prepared image as described in README.

Use conventional commits, actionable errors with wrapped causes and deterministic
machine-readable output. Docs and tests change with behavior.
