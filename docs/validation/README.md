# Validation

This directory holds the business-level validation harness and the results of
running it. The acceptance tests under `tests/` prove the runtime contract; this
harness proves that real projects work, and records what they cost.

## What is validated

Four fixtures of increasing architectural complexity. Every level is asserted
through the project's own interface, never through readiness:

| Level | Shape | What the assertions actually do |
| --- | --- | --- |
| `l1` | one HTTP service with a file-backed store | create/read/update/delete through HTTP, exact payload comparison, data survives `down`/`up`, `reset` wipes it, four parallel workspaces keep four distinct ports and cannot see each other's rows, `berth run` preserves argv verbatim |
| `l2` | frontend plus backend on two ports | the frontend renders live backend state, so it fails if the two services are not wired; wiping the backend's data through `reset` is visible in the frontend's page; two workspaces render their own backend only |
| `l3` | HTTP app plus a separate stateful SQL service | an idempotent migration owned by the service, real `INSERT`s over HTTP, SQL `SUM`/`COUNT`/`GROUP BY` aggregates compared against expected numbers, data survives `down`/`up`, a second workspace has its own database file and its own rows |
| `l4` | a real external project | the project's own build, typecheck and test commands run through `berth run` inside a workspace, and the first command must succeed in every parallel workspace |

`l3` uses SQLite inside a dedicated service process rather than PostgreSQL: the
runtimes available to the harness (a Windows laptop without a container engine,
GitHub runners) cannot host a database server on the native path. The database is
still a separate supervised process reached over TCP with real SQL, real
transactions and its own data directory, which is what the level is testing.

## What is measured

Every phase is timed with a monotonic clock and written to the JSON report:

- workspace creation (`new_ms`, `new_and_up_ms`, `new_migrate_up_ms`)
- readiness (`up_ms`, `up_again_ms`) and teardown (`down_ms`, `done_ms`)
- CLI cost per command (`status_ms`, `ports_ms`, `plan_ms`, `ls_ms`, `reset_ms`)
- command overhead (`run_via_berth_ms` against `run_plain_ms`, the same command
  executed directly in the same workspace)
- concurrency (`parallel4_new_and_up_wall_ms`, `parallel_all_workspaces_first_check_ms`)
- Copy on Write (`cow_48mb_new_ms`, `cow_writeback_ms`)
- resources (`supervised_rss_kb`, `workspace_disk_kb`, `container_mem_bytes`)
- level 4 project work (`clone_ms`, per-command `run_<name>_ms`)

## How to run it

Locally, native runtime, on whatever machine you have:

```sh
go build -o bin/berth ./cmd/berth
BERTH_HOME=/tmp/berth-home ./bin/berth doctor --fix      # pinned supervisor
python tests/validation/harness.py --berth ./bin/berth --mode native \
  --levels l1,l2,l3 --out native.json --markdown native.md
python tests/validation/harness.py --berth ./bin/berth --mode native \
  --levels l4 --l4-json tests/validation/l4-canvas.json --out l4.json --markdown l4.md
```

Container runtime, which needs a working Docker or Podman:

```sh
docker build -t berth-runtime:test -f runtime/Dockerfile .
python tests/validation/harness.py --berth ./berth --mode container \
  --image berth-runtime:test --levels l1,l2,l3 --out container.json --markdown container.md
```

`.github/workflows/validation.yml` runs the whole matrix: native on Linux and
Windows, and container on Linux with one container per workspace. It uploads the
JSON and markdown reports as artifacts.

## Project specs

`l4` is driven by a spec file so the same code validates a private business
repository locally and a public project in CI:

| Spec | Project | Used by |
| --- | --- | --- |
| `l4-public.json` | `gin-gonic/gin` | CI, both runtimes |
| `l4-canvas.json` | `qbox/canvas` (private) | local runs only; CI never touches it |

A spec contains no project code, only the repository URL, the setup hook, and
the commands to run.

## What these runs found

Running projects rather than probing readiness is what produced these, so they
are recorded here next to the numbers:

1. **A container workspace whose configuration declares no ports could not be
   used at all.** `berth new` created the container and every later command
   refused it with `runtime ownership/configuration mismatch`. The spec hash was
   computed from the workspace record, whose port maps serialise as `{}` while
   empty but read back as `null` after a round trip through `state.json`
   (`omitempty`), so the hash taken at creation never matched the hash taken at
   inspection. Fixed in `internal/runner` with a unit test that pins both
   directions; the container matrix passed level 4 afterwards.
2. **A workspace holding a heavy dependency tree could not be released.** After
   `git worktree remove` failed with `Directory not empty`, `berth done` refused
   to fall back to a recursive delete — the documented and intended behaviour —
   and the workspace was left unusable: `berth run` refused to enter it while
   `berth done` kept failing. The error now names the control files to remove,
   `docs/runtime.md` documents the recovery, and the harness performs it. A
   workspace with 1.5 GB of `node_modules` still needs that manual step; making
   the retry succeed on its own is worth a follow-up release.

One intermittent issue was seen once and did not reproduce on the next run:

- **`berth new` under four-way concurrency** failed on a Windows runner with
  `git worktree add -b berth/epsilon ... exit status 128`. The acceptance test
  exercises two workspaces in parallel and has never failed; the four-way
  fan-out belongs to this harness, so it is recorded as an observation with the
  exact error rather than as a confirmed defect. A repository-level lock around
  worktree creation may be what keeps it from becoming one.

Platform limitations that shape the numbers:

- On Windows a native process cannot receive a quoted argument or one containing
  a space, because the supervisor starts the command through `cmd /C` with the
  quotes escaped as data. `docs/runtime.md` documents the three forms that work;
  fixtures therefore read their configuration from the injected environment.
- Container numbers come from Linux runners and native numbers from both Linux
  runners and a Windows laptop, so cross-runtime comparisons should be read as
  orders of magnitude, not as a controlled benchmark.

## Results

The raw reports are committed next to this file. See `report.md` for the
summary, the environment of each run, and the limitations that apply to the
numbers.
