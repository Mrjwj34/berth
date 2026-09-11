#!/usr/bin/env bash
# Idempotent bootstrap for the lane development environment.
# Installs the toolchain required by AGENTS.md (Go 1.25+, just, process-compose)
# on top of Cursor's default base image, then warms the Go build/test caches.
set -euo pipefail

GO_VERSION="1.25.14"
JUST_VERSION="1.58.0"
PROCESS_COMPOSE_VERSION="v1.122.0"

log() { printf '\n=== %s ===\n' "$1"; }

# Go toolchain: go.mod pins Go 1.25, but the base image ships an older Go.
# Install a matching toolchain under /usr/local/go and expose it ahead of the
# distro Go via /usr/local/bin, which precedes /usr/bin on PATH.
install_go() {
  if [ -x /usr/local/go/bin/go ] && /usr/local/go/bin/go version | grep -q "go${GO_VERSION} "; then
    log "Go ${GO_VERSION} already installed"
  else
    log "Installing Go ${GO_VERSION}"
    local tarball="go${GO_VERSION}.linux-amd64.tar.gz"
    curl -fsSL -o "/tmp/${tarball}" "https://go.dev/dl/${tarball}"
    sudo rm -rf /usr/local/go
    sudo tar -C /usr/local -xzf "/tmp/${tarball}"
    rm -f "/tmp/${tarball}"
  fi
  sudo ln -sf /usr/local/go/bin/go /usr/local/bin/go
  sudo ln -sf /usr/local/go/bin/gofmt /usr/local/bin/gofmt
  hash -r
  go version
}

# just: task runner used by the repo's justfile.
install_just() {
  if command -v just >/dev/null 2>&1 && just --version | grep -q "${JUST_VERSION}"; then
    log "just ${JUST_VERSION} already installed"
  else
    log "Installing just ${JUST_VERSION}"
    local tarball="just-${JUST_VERSION}-x86_64-unknown-linux-musl.tar.gz"
    curl -fsSL -o "/tmp/${tarball}" \
      "https://github.com/casey/just/releases/download/${JUST_VERSION}/${tarball}"
    local dir="/tmp/just-${JUST_VERSION}"
    mkdir -p "${dir}"
    tar -C "${dir}" -xzf "/tmp/${tarball}"
    sudo install "${dir}/just" /usr/local/bin/just
    rm -rf "${dir}" "/tmp/${tarball}"
  fi
  just --version
}

# process-compose: runtime dependency lane shells out to for L2 process supervision.
install_process_compose() {
  if command -v process-compose >/dev/null 2>&1 && \
     process-compose version 2>/dev/null | grep -q "${PROCESS_COMPOSE_VERSION}"; then
    log "process-compose ${PROCESS_COMPOSE_VERSION} already installed"
  else
    log "Installing process-compose ${PROCESS_COMPOSE_VERSION}"
    local tarball="process-compose_linux_amd64.tar.gz"
    curl -fsSL -o "/tmp/${tarball}" \
      "https://github.com/F1bonacc1/process-compose/releases/download/${PROCESS_COMPOSE_VERSION}/${tarball}"
    local dir="/tmp/process-compose-${PROCESS_COMPOSE_VERSION}"
    mkdir -p "${dir}"
    tar -C "${dir}" -xzf "/tmp/${tarball}"
    sudo install "${dir}/process-compose" /usr/local/bin/process-compose
    rm -rf "${dir}" "/tmp/${tarball}"
  fi
  process-compose version 2>/dev/null | grep -E '^Version:' || true
}

install_go
install_just
install_process_compose

log "Warming Go module and build caches"
go mod download
go build -o bin/lane ./cmd/lane

log "Environment ready"
