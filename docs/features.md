# Features and User Guide

lane is a developer tool designed to manage parallel development environments on a local machine. It allows multiple software agents or human developers to work on separate tasks concurrently without port collisions, state corruption, or heavy virtual machine overhead.

## Core Concepts

### Workspaces

A workspace in lane combines an isolated Git worktree, a private data directory, dynamically allocated ports, and a set of supervised background processes. Each workspace checked out from Git uses its own branch configuration. Changes made to configuration files in one workspace do not affect other workspaces.

### Dual Runtime Modes

lane supports two distinct execution backends:

Native mode runs ordinary processes directly on the host operating system. It relies on environment variables and command line flags to assign dynamic ports and data paths. This mode offers maximum performance and zero container overhead.

Container mode runs a reusable Linux container per workspace using an existing local Docker or Podman installation. It preserves hardcoded loopback ports through internal forwarding gateways. If an application requires binding to port 8080 or port 5432, container mode isolates the network namespace while forwarding host traffic cleanly.

### Port Allocation

When a workspace declares named ports in its configuration, lane assigns unique host ports using an atomic reservation transaction. These assignments remain stable throughout the lifetime of the workspace. Programs access allocated ports through environment variables such as LANE_PORT_WEB for internal services and LANE_HOST_PORT_WEB for external host access.

### Private Data and State Isolation

Each workspace receives a dedicated data directory located at .lane/data in native mode and at /workspace/.lane/data in container mode. Databases, cache stores, and scratch files remain isolated to the active workspace. Writable dependencies can be copied using Copy on Write on supported filesystems such as Linux reflink and macOS clonefile to avoid physical disk duplication.

### Daemonless Operation

lane does not run a continuous background daemon. Instead, it relies on file locks and a single machine-level registry file. Long operations hold individual workspace locks rather than a global machine lock. Process supervision is delegated to process-compose or the container runtime.

## Workspace Management

### Creating Workspaces

Use the new command to create an isolated worktree and allocate required resources:

```sh
lane new feature-login --up
```

The up flag immediately prepares the workspace, runs setup hooks, and starts all declared background processes.

### Adopting Existing Worktrees

If you already created a Git worktree outside lane, register it with the adopt command:

```sh
lane adopt
```

lane assigns ports and metadata without taking destructive ownership. Adopted worktrees are never deleted automatically by cleanup commands.

### Running Commands

The run command executes commands within the execution context of the workspace:

```sh
lane run -- npm test
```

Environment variables for assigned ports, data directories, and workspace roots are injected automatically. Arguments are passed directly to the target executable without shell reinterpretation.

### Resetting Workspaces

When test data becomes dirty or invalid, reset restores the private data directory and re-runs setup hooks:

```sh
lane reset feature-login
```

lane records the reset intent to disk before modifying files. If interrupted, the reset is resumed cleanly on the next command.

### Stopping and Cleaning Up

To stop running background processes while preserving workspace data:

```sh
lane down feature-login
```

To tear down a completed workspace:

```sh
lane done feature-login
```

lane verifies that all local commits are either merged or pushed to upstream before removing files. Primary worktrees and external checkouts are protected from accidental removal.

### Conservative Garbage Collection

The gc command cleans up orphaned state entries and expired workspaces:

```sh
lane gc --dry-run
lane gc
```

Garbage collection operates strictly with non-destructive rules. Workspaces containing unpushed commits or active commands are skipped.

## Integration with Coding Agents

lane includes embedded skills and hooks for modern coding tools including Claude Code and Cursor.

Run init to install skill definitions into your repository:

```sh
lane init
```

Run hook install to link worktree lifecycle events with your editor:

```sh
lane hook install all
```

Agents can query workspace status and port contracts in structured JSON format using flags such as --json.
