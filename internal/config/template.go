package config

const Template = `# lane.yaml — isolation primitives only. lane does not know any datastore.
# Agent onboarding: read SKILL.md, declare how the project already listens,
# then: lane new smoke --up
# and commit this file once the smoke workspace is healthy.
version: 1
base: main                                   # baseline branch; created branches use prefix lane/

# General isolation: each workspace gets a private network. Processes keep
# their own listen ports; lane discovers them and publishes unique host ports.
# isolate: net
#
# Optional names for those host ports (LANE_PORT_API). listen: is a hint, not
# a requirement — omit it and lane still publishes whatever is listening.
# ports:
#   api:
#     listen: 8080
#
# Short form ports: [web, api] only allocates LANE_PORT_* for processes that
# already accept --port / $PORT (no network isolation).

# env:
#   DATABASE_URL: postgres://127.0.0.1:${LANE_PORT_PG}/app

# Optional dotenv managed block (# BEGIN LANE ... # END LANE)
# env_file: .env.local

# Fast clone/hardlink/copy from the main worktree after git checkout.
# copy_dirs:
#   - node_modules

hooks:
  setup: []
  teardown: []

# process-compose syntax, passed through. Commands see every LANE_* variable.
# processes:
#   api:
#     command: go run ./cmd/server
#     readiness_probe:
#       http_get:
#         port: 8080
#         path: /healthz

gc:
  idle_stop_hours: 4
  remove_after_days: 7
  max_workspaces: 8
`
