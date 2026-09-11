# Recipe: Postgres as a process

## When to use

The project needs a local Postgres per workspace. lane has no Postgres type — declare `postgres` like any other process, bind it to `LANE_PORT_PG`, and keep files under `LANE_DATA_DIR`.

If Postgres already binds `5432`, declare `listen: 5432` and keep the command unchanged. If the binary already accepts `-p`, you may pass `$LANE_PORT_PG` instead. Discover the host-published port with `lane ports --json`.

## `lane.yaml` fragment

```yaml
ports: [pg]
env:
  DATABASE_URL: postgres://localhost:${LANE_PORT_PG}/app
hooks:
  setup:
    - mkdir -p "$LANE_DATA_DIR"
    - test -d "$LANE_DATA_DIR/pg" || initdb -D "$LANE_DATA_DIR/pg" --no-locale --encoding=UTF8
processes:
  pg:
    command: postgres -D $LANE_DATA_DIR/pg -p $LANE_PORT_PG -k $LANE_DATA_DIR
    readiness_probe:
      exec:
        command: pg_isready -h 127.0.0.1 -p "$LANE_PORT_PG"
```

`-k $LANE_DATA_DIR` keeps the Unix socket inside the workspace data dir so parallel workspaces do not collide on `/tmp`.

Optional: derive `DATABASE_URL` into an `env_file` managed block if frontend or ORM tools only read a dotenv file. After `lane new smoke --up`, confirm `pg_isready` via `lane status --json` before running migrations with `lane run --`.
