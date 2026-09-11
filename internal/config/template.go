package config

const Template = `# Commit berth.yaml before creating workspaces; each worktree uses its own config.
version: 1
base: main
runtime:
  backend: native
# Native mode: programs must accept a configurable port/data path.
# ports: [web]
# env:
#   PORT: ${BERTH_PORT_WEB}
# processes:
#   web:
#     command: npm run dev -- --port "$BERTH_PORT_WEB"

# For hardcoded ports, prepare a Linux runtime image once, then use:
# runtime:
#   backend: container
#   engine: docker
#   image: berth-runtime:local
# ports: [web]
# listen: {web: 8080}
# processes:
#   web:
#     command: python3 -m http.server 8080 --bind 127.0.0.1
#     readiness_probe:
#       http_get: {host: 127.0.0.1, port: 8080, path: /}

# BERTH_PORT_* is relative to the execution context.
# BERTH_HOST_PORT_* always refers to the host publication (e.g. browser URLs).
# env_file: .env.local
# copy_dirs: [node_modules]  # CoW where supported, independent copy otherwise
hooks:
  setup: []                # must be idempotent; incomplete setup is retried
  teardown: []
gc:
  idle_stop_hours: 0        # explicit opt-in; based on berth operations
  remove_after_days: 0      # GC is invoked explicitly, never forced
  max_workspaces: 0
`
