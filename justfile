[private]
default:
    @just --list

set shell := ["sh", "-cu"]
set windows-shell := ["powershell.exe", "-NoLogo", "-Command"]

bin-name := if os() == "windows" { "bin/berth.exe" } else { "bin/berth" }

# Run format and static checks
check: _go-format-check
    go vet ./cmd/... ./internal/...
    go test ./cmd/... ./internal/...

# Build the CLI binary
build: _ensure-bin
    go build -o {{bin-name}} ./cmd/berth

# Run tests with race detection
test:
    go test -race ./cmd/... ./internal/...

# Format Go sources
fmt:
    gofmt -w cmd internal

[unix]
[private]
_ensure-bin:
    mkdir -p bin

[windows]
[private]
_ensure-bin:
    New-Item -ItemType Directory -Force -Path bin | Out-Null

[unix]
[private]
_go-format-check:
    #!/usr/bin/env sh
    unformatted=$(gofmt -l cmd internal)
    test -z "$unformatted" || { printf '%s\n' "$unformatted"; exit 1; }

[windows]
[private]
_go-format-check:
    $unformatted = gofmt -l cmd internal; if ($unformatted) { $unformatted; exit 1 }
