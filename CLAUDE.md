# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

`try-go` is a Go reimplementation of [`tobi/try`](https://github.com/tobi/try) — an ephemeral workspace manager that creates date-prefixed directories for temporary git clones, worktrees, and scratch work. It outputs shell scripts to stdout that get eval'd by a shell wrapper function.

## Build & Development Commands

```bash
# Setup (install Go, revive, goreleaser via mise)
mise trust && mise install

# Build
mise run build          # or: go build -o try ./cmd/try

# Test
go test ./...           # run all tests
go test ./internal/try  # run package tests
go test ./internal/try -run TestFuzzyScore  # run a single test

# Lint
mise run lint           # runs: go vet ./... && revive -set_exit_status ./...

# Format
mise run format         # or: go fmt ./...
```

## GitHub Actions Rule

When you modify files under `.github/workflows/`, run `pinact` afterward to pin action versions.

```bash
mise x pinact -- pinact run
```

## Architecture

All code lives in two packages:

- **`cmd/try/main.go`** — Minimal entrypoint, calls `trypkg.Main()`
- **`internal/try`** (package `trypkg`) — All implementation

### Key files in `internal/try`

| File | Purpose |
|------|---------|
| `main.go` | CLI parsing via `kong`, command routing (`init`, `clone`, `worktree`, `exec`), arg normalization |
| `selector.go` | Interactive TUI using `bubbletea`/`lipgloss` with fuzzy search, rename (Ctrl-R), multi-delete (Ctrl-D) |
| `gitutil.go` | Git URL parsing (HTTPS/SSH), generates dated directory names for clones |
| `pathutil.go` | Path expansion (`~/`), name sanitization, unique directory name generation |
| `scriptutil.go` | Shell script generation (cd, mkdir, clone, worktree, rename, delete) for bash/fish |

### Design pattern: script output

The binary does NOT execute shell commands directly. Instead, it prints shell scripts to stdout. The shell integration (`try init`) creates a wrapper function that evals this output. TUI rendering goes to stderr so it doesn't interfere with the eval'd stdout.

### Hidden test flags

`--and-type`, `--and-keys`, `--and-exit`, `--and-confirm` are hidden flags used to simulate user input in integration tests. They are extracted before kong parsing via `extractCompatFlags()`.

## Testing

Tests are in `internal/try/*_test.go` using the same package (`trypkg`). Integration tests use `t.TempDir()` for isolation and `--and-keys` to simulate keyboard input. Key patterns:

- `compat_spec_test.go` / `compat_more_test.go` — Integration tests for full exec/clone/worktree flows
- `compat_tui_delete_test.go` — TUI deletion flow tests with simulated input
- `selector_test.go` — Fuzzy scoring unit tests
- Date-dependent paths use regex matching (e.g., `^\d{4}-\d{2}-\d{2}-name$`)

## Dependencies

- `kong` — CLI argument parsing
- `bubbletea` — TUI framework (Elm architecture)
- `lipgloss` — Terminal styling

## Release

GoReleaser builds for linux/amd64, darwin/arm64, windows/amd64. Version info is injected via ldflags into `internal/try` package vars (`version`, `commit`, `shortCommit`). Releases auto-update the Homebrew tap at `upamune/homebrew-tap`.
