<table>
  <tr>
    <td width="140" align="center" valign="middle">
      <img src="https://cdn.jwjbox.dev/lane.png" alt="lane logo" width="120" />
    </td>
    <td valign="middle">
      <p><strong>Fast, low-cost management of parallel development environments</strong></p>
      <p>
        <a href="https://github.com/Mrjwj34/lane/actions/workflows/ci.yml"><img src="https://github.com/Mrjwj34/lane/actions/workflows/ci.yml/badge.svg" alt="CI Status" /></a>
        <a href="https://github.com/Mrjwj34/lane/releases"><img src="https://img.shields.io/github/v/release/Mrjwj34/lane" alt="Latest Release" /></a>
        <a href="https://golang.org"><img src="https://img.shields.io/github/go-mod/go-version/Mrjwj34/lane" alt="Go Version" /></a>
        <a href="LICENSE"><img src="https://img.shields.io/github/license/Mrjwj34/lane" alt="License" /></a>
      </p>
      <p>
        <a href="README.zh-CN.md">简体中文</a> | English
      </p>
    </td>
  </tr>
</table>

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

### 1. Install lane

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

### 2. Initialize your project

Run init in the root of your Git repository:

```sh
lane init
lane hook install all
```

This installs the skill definitions into your agent directories for Claude Code and Cursor, and sets up editor hooks.

### 3. Let your Agent configure the workspace

Prompt your coding agent:

> Inspect this repository and configure lane.yaml based on existing startup scripts, toolchains, and listen ports.

The agent reads the embedded skill, inspects project scripts and configuration files, and declares appropriate ports and processes in lane.yaml.

If you prefer manual configuration, edit lane.yaml directly:

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

### 4. Create and run workspaces

Prompt your agent to work in an isolated environment, or run the command directly:

```sh
lane new feature-a --up
```

View active workspaces:

```sh
lane ls --json
lane status feature-a
```

Run tests or commands within the workspace environment:

```sh
lane run -- npm test
```

When work is finished and commits are pushed:

```sh
lane done feature-a
```

## Detailed Documentation

- User Guide: [docs/features.md](docs/features.md) introduces practical workflows, data isolation, and agent integrations.
- Runtime contract: [docs/runtime.md](docs/runtime.md) covers networking, port allocation, container behavior, and lifecycle guarantees.
- Architecture decisions: [docs/architecture.md](docs/architecture.md) explains the daemonless lock model, state recovery, and safety invariants.
