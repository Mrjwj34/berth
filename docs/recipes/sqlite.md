# Recipe: SQLite (no process)

## When to use

The project can use a file-backed SQLite database. lane has no SQLite type and starts no database process — point the app at a file under `LANE_DATA_DIR`. That file is private to the worktree and is destroyed with the workspace.

No datastore port is required. If the app itself listens on a hardcoded port, declare `listen:` — do not rewrite the app. Discover the host-published port with `lane ports --json`.

## `lane.yaml` fragment

```yaml
ports: [api]
env:
  DATABASE_URL: sqlite://${LANE_DATA_DIR}/app.db
  PORT: ${LANE_PORT_API}
hooks:
  setup:
    - mkdir -p "$LANE_DATA_DIR"
processes:
  api:
    command: "PORT=$LANE_PORT_API SQLITE_PATH=$LANE_DATA_DIR/app.db go run ./cmd/server"
    readiness_probe:
      http_get:
        port: "${LANE_PORT_API}"
        path: /healthz
```

`$LANE_DATA_DIR/app.db` is created by the app or a setup/migration command (`lane run --`). There is no `sqlite` process to probe — probe the application, not the file.
