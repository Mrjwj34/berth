# Recipe: Elasticsearch as a process

## When to use

The project needs a local Elasticsearch per workspace. lane has no Elasticsearch type — declare the `elasticsearch` binary as a process, set `http.port` from `LANE_PORT_ES`, and keep `path.data` / `path.logs` under `LANE_DATA_DIR`.

If Elasticsearch already binds `9200`, declare `listen: 9200` and keep the command unchanged. If the binary already accepts `http.port`, you may pass `$LANE_PORT_ES` instead. Discover the host-published port with `lane ports --json`.

## `lane.yaml` fragment

```yaml
ports: [es]
env:
  ELASTICSEARCH_URL: http://127.0.0.1:${LANE_PORT_ES}
hooks:
  setup:
    - mkdir -p "$LANE_DATA_DIR/es" "$LANE_DATA_DIR/es-logs"
processes:
  es:
    command: elasticsearch -E path.data=$LANE_DATA_DIR/es -E path.logs=$LANE_DATA_DIR/es-logs -E http.port=$LANE_PORT_ES -E http.host=127.0.0.1 -E discovery.type=single-node -E xpack.security.enabled=false
    readiness_probe:
      http_get:
        port: "${LANE_PORT_ES}"
        path: /_cluster/health
```

Single-node + security disabled is for local Agent workspaces only. Heap and disk use are large; prefer `lane down` when idle and rely on `gc.idle_stop_hours` so unused ES processes do not linger.
