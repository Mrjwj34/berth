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

## Results

The raw reports are committed next to this file. See `report.md` for the
summary, the environment of each run, and the limitations that apply to the
numbers.
