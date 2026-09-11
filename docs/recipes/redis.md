# Recipe: Redis as a process

## When to use

The project needs a local Redis per workspace. lane has no Redis type — declare `redis-server` as a process, bind `--port` to `LANE_PORT_REDIS`, and keep `--dir` under `LANE_DATA_DIR`.

The app must read the port from env (`REDIS_URL` / `LANE_PORT_REDIS`). Never hardcode `6379`. Discover the assigned port with `lane ports --json` or `lane status --json`.

## `lane.yaml` fragment

```yaml
ports: [redis]
env:
  REDIS_URL: redis://127.0.0.1:${LANE_PORT_REDIS}
hooks:
  setup:
    - mkdir -p "$LANE_DATA_DIR/redis"
processes:
  redis:
    command: redis-server --port "$LANE_PORT_REDIS" --dir "$LANE_DATA_DIR/redis" --bind 127.0.0.1 --daemonize no --protected-mode yes --save ""
    readiness_probe:
      exec:
        command: redis-cli -h 127.0.0.1 -p "$LANE_PORT_REDIS" ping
```

`--daemonize no` keeps the process in the foreground so lane can supervise it. Persistence files stay in `$LANE_DATA_DIR/redis` and disappear with the workspace.
