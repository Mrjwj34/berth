# Linux runtime / Docker

Prefer runtime.backend: container when a workspace needs unchanged hardcoded
ports. A single reusable container carries the workspace's ordinary processes;
process-compose runs inside it. This avoids confusing a short-lived docker CLI
process with the daemon-owned lifetime of the actual container.

Build runtime/Dockerfile once in the lane repository, or use it as a basis for a
project toolchain image. Select that existing image in lane.yaml:

```yaml
version: 1
runtime:
  backend: container
  engine: docker
  image: lane-runtime:local
ports: [web]
listen: {web: 8080}
processes:
  web:
    command: python3 -m http.server 8080 --bind 127.0.0.1
```

Services, probes, hooks and lane run share this runtime. Public host endpoints use
LANE_HOST_PORT_WEB; in-runtime clients keep localhost:8080. No docker.sock or host
credentials are automatically mounted. Do not use --network=host or guessed ports
to circumvent the workspace contract. See ../runtime.md for supported protocols,
rootless/desktop engine caveats and ownership/cleanup rules.
