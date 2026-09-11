# Recipe: Docker is just a process

## When to use

The project already has a container image and you want that container bound to this workspace's ports and (optionally) data dir. lane does not know Docker. `docker run --rm --name ... -p $LANE_PORT_X:...` is an ordinary `processes` command, same as a native binary.

Map the workspace host port on the left of `-p` (`$LANE_PORT_X:containerPort`). The container may keep its own listen port. Discover the host-published port with `lane ports --json`. Prefer a native binary (see the other recipes) when the tool is already installed — Docker is optional, not a lane feature.

## `lane.yaml` fragment

```yaml
ports: [pg]
env:
  DATABASE_URL: postgres://postgres@127.0.0.1:${LANE_PORT_PG}/app
hooks:
  setup:
    - mkdir -p "$LANE_DATA_DIR/pg"
processes:
  pg:
    command: docker run --rm --name "lane-${LANE_SLUG}-pg" -p "$LANE_PORT_PG:5432" -v "$LANE_DATA_DIR/pg:/var/lib/postgresql/data" -e POSTGRES_HOST_AUTH_METHOD=trust -e POSTGRES_DB=app postgres:16
    readiness_probe:
      exec:
        command: pg_isready -h 127.0.0.1 -p "$LANE_PORT_PG"
```

`--rm` and a name that includes `$LANE_SLUG` keep containers from colliding across workspaces. Map the workspace port on the left of `-p` (`$LANE_PORT_X:containerPort`). Persist container files with a bind mount into `$LANE_DATA_DIR`, not a Docker named volume shared by every workspace.

lane still supervises this as one process. Do not call `docker compose` / process-compose yourself; use `lane up`, `lane down`, `lane logs pg`, and `lane doctor` on failure.
