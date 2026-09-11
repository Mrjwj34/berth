# Recovery reference

Read this when a lane command fails or a workspace is in an unknown state.

## Errors and what they mean

| Message | Meaning | Action |
| --- | --- | --- |
| `workspace not found; run lane ls or lane adopt` | The current directory is not inside a registered workspace and no slug was given. | `cd` into the path `lane new` printed, or pass the slug. |
| `port contract changed; create a new workspace` | `lane.yaml` declares different ports or runtime settings than the workspace was created with. | Create a new workspace from the updated branch; in-place edits are refused on purpose. |
| `worktree is dirty; commit changes first` | `lane done` refuses to remove a checkout with uncommitted work. | Commit or discard the changes, then retry. |
| `work is not preserved: …` | The branch has commits neither merged into the base nor present upstream. | Push or merge the branch, then retry `lane done`. |
| `refusing to operate on the primary worktree` | `lane adopt` or `lane done` was pointed at the main checkout. | Only linked worktrees are candidates; create one with `lane new`. |
| `checkout is on a detached HEAD` | `lane adopt` needs a branch, and harness-created worktrees are often detached. | Check out a branch there, or create `lane new <slug>` instead. |
| `process state unknown; inspect …` | The supervisor socket or port file exists but does not answer. | Preserve the data, read `.lane/process-compose.log` and `lane logs <proc>`, and leave `.lane/` in place. |
| `processes not ready` | A readiness probe never passed inside the startup window. | `lane status --json` for the failing process, `lane logs <proc>` for its output, then fix the command or the probe. |
| `process <name> failed with exit <n>` | The process exited during startup. | Read its logs; the worktree is kept so the command can be fixed and retried. |
| `lane doctor` FAIL lines | git, engine, image, process-compose or state is missing or unhealthy. | `lane doctor --fix` installs native dependencies such as process-compose; a container engine or image is prepared once by you, never pulled implicitly. |

## After a failed setup

The workspace keeps its worktree, data and state, and `lane new <same-name>` or
`lane up` retries setup once the configuration or hook is fixed. Setup hooks must
be idempotent for that retry to be safe.

## What lane gc collects

`lane gc` is always explicit and never forced, and it revalidates every candidate
under the workspace lock. Candidates are: a registration whose directory is gone,
an idle runtime past `gc.idle_stop_hours`, a stopped and merged workspace older
than `gc.remove_after_days`, and workspaces above `gc.max_workspaces` for that
repository. Removal still requires preserved commits and skips active operations;
`lane gc --dry-run --json` previews the list. Use `lane down` to keep a workspace
and its data, `lane done` to release it.

## Ownership

`lane done` never removes an adopted checkout: it stops the runtime and
unregisters the workspace, preserving directory, data and branch even with
`--force`. `--force` cannot bypass ownership, identity, primary-worktree or
shutdown checks, so when one of them refuses, fix the underlying condition instead
of escalating.
