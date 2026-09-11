<p>
  <a href="https://github.com/Mrjwj34/berth">
    <img src="https://cdn.jwjbox.dev/berth.png" alt="berth logo" align="left" width="110" style="margin-right: 20px; margin-bottom: 12px;" />
  </a>
  <span style="font-size: 1.5em; font-weight: bold; line-height: 1.3;">Fast, low-cost management of parallel development environments</span><br><br>
  <a href="https://github.com/Mrjwj34/berth/actions/workflows/ci.yml"><img src="https://github.com/Mrjwj34/berth/actions/workflows/ci.yml/badge.svg" alt="CI Status" /></a>
  <a href="https://github.com/Mrjwj34/berth/releases"><img src="https://img.shields.io/github/v/release/Mrjwj34/berth" alt="Latest Release" /></a>
  <a href="https://golang.org"><img src="https://img.shields.io/github/go-mod/go-version/Mrjwj34/berth" alt="Go Version" /></a>
  <a href="LICENSE"><img src="https://img.shields.io/github/license/Mrjwj34/berth" alt="License" /></a>
</p>
<br clear="left" />

<p align="center">
  English | <a href="README.zh-CN.md">简体中文</a>
</p>

berth provisions lightweight, isolated local workspaces for parallel agent programming, combining Git worktrees, private data directories, dynamic port assignment, and supervised processes without virtual machine overhead. Every workspace is a self-contained development environment — its own branch, its own data, its own ports, its own supervised processes — created in milliseconds, with no background daemon.

<p align="center">
  <img src="https://cdn.jwjbox.dev/berth-demo.gif" alt="berth creates an isolated workspace and starts its declared services" width="760" />
</p>

## Install

Choose any of the following installation methods.

### Download prebuilt binaries

Prebuilt binaries for Linux, macOS, and Windows are available on the GitHub Releases page. Extract the archive and place the executable in your PATH.

### Install with Go

Requires Go 1.25 or newer:

```sh
go install github.com/Mrjwj34/berth/cmd/berth@latest
```

### Build from source

```sh
git clone https://github.com/Mrjwj34/berth.git
cd berth
go build -o bin/berth ./cmd/berth
```

## 30-Second Quick Start

### 1. Initialize berth in your repository

Run initialization in your repository root:

```sh
berth init
berth hook install all
```

This generates a base template, installs the agent skill to `.agents/skills/berth` for Cursor, Codex, pi and Antigravity, and merges the Cursor worktree adapter into `.cursor/worktrees.json`.

### 2. Let your Agent configure berth.yaml

Prompt your coding agent:

> Inspect this repository and configure berth.yaml based on existing startup scripts, toolchains, and listen ports.

The agent reads the embedded skill, inspects project dependencies, and declares required ports and services automatically.

If you prefer to configure manually, see the complete configuration guide and examples in [docs/features.md](docs/features.md).

### 3. Create an isolated workspace and start services

```sh
berth new feature-a --up
```

### 4. Run tests within the workspace context

```sh
berth run -- npm test
```

### 5. Clean up after work is pushed

```sh
berth done feature-a
```

## Why parallel coding agents collide

When multiple software agents work on the same repository in parallel, simple branches are not enough. Running concurrent test suites or background servers immediately causes TCP port collisions, database state corruption, and leaked processes.

Developers often face an awkward trade-off. Raw Git worktrees manage file trees but leave ports, databases, and background processes entirely unhandled. Manual port assignment leads to accidental commits of local port overrides. Full container stacks consume excessive memory, introduce filesystem performance penalties on macOS and Windows, and slow down agent feedback loops.

berth solves this by providing a unified workspace abstraction. Each workspace receives its own linked Git checkout, private data directory, atomic port reservations, and process supervision. Operations complete in milliseconds without requiring a persistent background daemon.

## How berth compares

| | raw `git worktree` | devcontainer / Docker | **berth** |
| --- | --- | --- | --- |
| Isolation boundary | files and branches | the whole machine or VM | the workspace: worktree, private data, ports |
| Create a new environment | instant, source files only | image build and container start | milliseconds, no daemon |
| Port allocation | manual, leaks into commits | container network | atomic reservation, `BERTH_PORT_*` |
| Data isolation | none | volumes and bind mounts | `.berth/data` per workspace |
| Process supervision | none | inside the container | process-compose or the container runtime |
| Host filesystem speed | native | bind-mount penalty on macOS and Windows | native, Copy on Write where available |

### Git worktree alone

Git worktree provides isolated branches and checkouts, but offers no assistance with port allocation, database persistence, process supervision, or environment variables. Everything beyond source files must be managed manually.

### Docker and devcontainer

Docker and devcontainer provide complete operating system isolation. However, they come with substantial startup latency, heavy memory footprints, bind mount filesystem slowdowns on non-Linux hosts, and repetitive image rebuild overhead. They treat the entire machine environment as the boundary of isolation. Reach for them when you genuinely need a different kernel or a toolchain your host cannot provide.

### berth

berth chooses a pragmatic middle path. In native mode, it runs host processes directly with zero virtualization overhead while isolating ports, data directories, and environment variables. In container mode, it reuses a single Linux container per workspace without per-command rebuilds, forwarding host traffic through loopback gateways. berth treats the workspace rather than the entire operating system as the primary unit of isolation.

## Agent integration

berth is a plain CLI, so any agent that can run shell commands can drive it. `berth init` additionally installs one skill to `.agents/skills/berth/SKILL.md`, the single location read by Cursor, Codex, pi and Antigravity — there is exactly one copy to keep current, and berth writes nothing into `AGENTS.md`, `GEMINI.md` or any harness's own rules file.

`berth hook install cursor` merges a Cursor worktree adapter into `.cursor/worktrees.json`, so `berth adopt --setup` runs inside every worktree Cursor creates in the Agents Window, the IDE or the CLI. Session-end hooks are deliberately not installed: stop work explicitly with `berth down` (keep data) or `berth done` (release the workspace), and let `berth gc` reclaim what was abandoned.

## FAQ

**Can two agents work on the same repository at the same time?**

Yes. Each workspace is its own linked Git worktree on branch `berth/<slug>`, created beside the repository at `<repo>.berths/<slug>`, with its own private data directory and its own dynamically allocated ports. The primary checkout is never a workspace, so it stays usable while agents work in parallel.

**How does each agent get its own port?**

Declare the named ports your project needs in `berth.yaml`, for example `ports: [web, pg]`. berth reserves a unique host port for every workspace in one atomic, cross-process transaction, and the assignment stays stable for the lifetime of that workspace. Programs read `BERTH_PORT_WEB` for the internal port and `BERTH_HOST_PORT_WEB` to reach the service from a host browser.

**Why not just use devcontainers or Docker?**

Use them when you need a different kernel, full OS isolation, or a toolchain your host cannot provide. The trade-off is startup latency, memory footprint, bind-mount filesystem penalties on macOS and Windows, and rebuild overhead. berth's native mode runs ordinary host processes with zero virtualization overhead, while its container mode covers projects that hardcode loopback ports by reusing one Linux container per workspace with no per-command image builds.

**Does berth run a daemon?**

No. berth relies on file locks and a single machine-level registry file. Long operations hold an individual workspace lock rather than a global machine lock, and process supervision is delegated to process-compose or the container runtime.

**What happens to my work when a workspace goes away?**

`berth down <slug>` gracefully stops the processes and keeps all data. `berth done <slug>` verifies the work is preserved before removing the workspace and its worktree. `berth reset <slug>` wipes only the private data directory and reruns the setup hooks. `berth gc` reclaims abandoned workspaces and never uses force.

**Does it work on Windows and macOS?**

Yes. Native binaries are published for Linux, macOS and Windows on both amd64 and arm64. Container mode requires an existing local Docker or Podman installation. Writable dependency copies use Copy on Write where the filesystem supports it — reflink on Linux btrfs and xfs, clonefile on APFS — and never silently fall back to hardlinks.

## Architecture

berth separates workspace management from the execution context. Each workspace has its own branch configuration, runtime plan, and operation lock.

<div align="center">
  <img src="https://cdn.jwjbox.dev/berth-architecture.png" alt="berth architecture" width="800" />
</div>

Detailed documentation:
- User Guide: [docs/features.md](docs/features.md) covers complete configuration references, practical workflows, and agent integrations.
- Runtime contract: [docs/runtime.md](docs/runtime.md) explains networking, port mapping, and container execution rules.
- Architecture decisions: [docs/architecture.md](docs/architecture.md) details the daemonless lock model, state recovery, and safety invariants.
