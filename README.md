<div align="center">
  <img src="https://cdn.jwjbox.dev/lane.png" alt="lane logo" width="120" />
  <h1>lane</h1>
  <p>Fast, low-cost management of parallel development environments</p>
  <p>
    <a href="https://github.com/Mrjwj34/lane/actions/workflows/ci.yml"><img src="https://github.com/Mrjwj34/lane/actions/workflows/ci.yml/badge.svg" alt="CI Status" /></a>
    <a href="https://github.com/Mrjwj34/lane/releases"><img src="https://img.shields.io/github/v/release/Mrjwj34/lane" alt="Latest Release" /></a>
    <a href="https://golang.org"><img src="https://img.shields.io/github/go-mod/go-version/Mrjwj34/lane" alt="Go Version" /></a>
    <a href="LICENSE"><img src="https://img.shields.io/github/license/Mrjwj34/lane" alt="License" /></a>
  </p>
  <p>
    <a href="README.zh-CN.md">简体中文</a> | English
  </p>
</div>

## Features

- Independent workspaces using Git worktrees with separate data directories, runtime configuration, and port allocations
- Native execution mode running ordinary host processes with dynamic port assignment and process supervision
- Container execution mode running a reusable Linux container per workspace with loopback forwarding to preserve hardcoded ports
- Daemonless architecture relying on local file locks and state persistence without a long-running background service
- Safe lifecycle protection preventing removal of primary checkouts, preserving unpushed commits, and safeguarding adopted workspaces
- Built-in Agent skills and workflow integration for tools like Claude Code and Cursor

## Architecture

lane separates workspace management from the execution context. Each workspace has its own branch configuration, runtime plan, and lifecycle lock.

<div align="center">
  <img src="https://cdn.jwjbox.dev/lane-architecture.png" alt="lane architecture" width="800" />
</div>

## Quick Start

### Installation

Download prebuilt binaries from GitHub Releases:

Linux, macOS, and Windows binaries are available on the releases page. Extract the archive and place the executable in your PATH.

Install using Go:

```sh
go install github.com/Mrjwj34/lane/cmd/lane@latest
```

Build from source:

```sh
git clone https://github.com/Mrjwj34/lane.git
cd lane
go build -o bin/lane ./cmd/lane
```

### Initialize a project

Run init in the root of your Git repository:

```sh
lane init
```

This creates a default lane.yaml and installs the skill file into your agent directories. If you use Cursor or Claude Code worktrees, install the hooks:

```sh
lane hook install all
```

### Configure lane.yaml

Define the base branch, exposed ports, and managed processes:

```yaml
version: 1
base: main
ports: [web]
env:
  PORT: ${LANE_PORT_WEB}
processes:
  web:
    command: npm run dev -- --port "$LANE_PORT_WEB"
    readiness_probe:
      http_get: {host: 127.0.0.1, port: "${LANE_PORT_WEB}", path: /}
```

### Manage workspaces

Create and start a new workspace:

```sh
lane new feature-a --up
```

Inspect active workspaces and allocated ports:

```sh
lane ls --json
lane status feature-a
```

Run one-off commands inside the workspace context:

```sh
lane run -- npm test
```

Stop processes when paused:

```sh
lane down feature-a
```

Safely remove the workspace when work is pushed or merged:

```sh
lane done feature-a
```

## Detailed Documentation

- Runtime contract: [docs/runtime.md](docs/runtime.md) covers networking, port allocation, container behavior, and lifecycle guarantees.
- Architecture decisions: [docs/architecture.md](docs/architecture.md) explains the daemonless lock model, state recovery, and safety invariants.
