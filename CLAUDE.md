# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

Agent Deck is a terminal session manager (TUI + CLI) for AI coding agents. It provides a "mission control" interface for managing multiple AI sessions (Claude Code, Gemini CLI, OpenCode, Codex, Cursor, custom tools) running in tmux. Written in Go 1.24+ using the Bubble Tea TUI framework.

## Build & Development Commands

```bash
make build          # Build binary to ./build/agent-deck
make test           # Run all tests: go test -v ./...
make lint           # Run golangci-lint
make fmt            # Format code: go fmt ./...
make run            # Run directly: go run ./cmd/agent-deck
make dev            # Auto-reload dev mode (uses air)
make install-user   # Install to ~/.local/bin
```

Run a single test:
```bash
go test -v -run TestFunctionName ./internal/session/
```

Run a single package's tests:
```bash
go test -v ./internal/tmux/
```

Debug mode: `AGENTDECK_DEBUG=1 agent-deck` (logs to `~/.agent-deck/debug.log`)

## Architecture

**Entry point**: `cmd/agent-deck/main.go` — subcommand dispatch and TUI launch.

**Core packages**:
- `internal/ui/` — TUI layer using Bubble Tea (Elm architecture: Model-Update-View). `home.go` is the root model composing all dialogs and panels.
- `internal/session/` — Core business logic. Session lifecycle (`instance.go`), JSON persistence (`storage.go`), profile/config management, tool integrations (Claude, Gemini), MCP catalog, global search, group tree.
- `internal/tmux/` — tmux integration. Session cache, pane capture, activity watching, tool-specific prompt detection (`detector.go`).
- `internal/mcppool/` — MCP socket pooling for shared processes across sessions.
- `internal/git/` — Worktree management and repo detection.

**Data flow**: User config lives in `~/.agent-deck/config.toml` (TOML, parsed by `session/userconfig.go`). Session data persists to `~/.agent-deck/profiles/<profile>/sessions.json` with 3 rolling backups.

## Key Patterns

**Version management**: `const Version` in `cmd/agent-deck/main.go` must match the git tag for releases. Bump with `chore: bump version to v0.8.XX`.

**tmux naming**: All tmux sessions use prefix `agentdeck_` (constant `SessionPrefix` in `tmux/tmux.go`).

**Flag parsing**: `reorderArgsForFlagParsing()` in `main.go` moves positional args to the end because Go's `flag` package stops at the first non-flag arg.

**Status detection**: `tmux/detector.go` uses tool-specific prompt regex patterns to detect running/waiting/idle status, refreshed via cached `tmux list-windows` every 2 seconds.

## Testing Safety

Every test package with tmux interaction has a `testmain_test.go` that forces `AGENTDECK_PROFILE=_test` to prevent overwriting production data, and cleans up orphaned test tmux sessions. This pattern exists in `internal/session/`, `internal/tmux/`, and `cmd/agent-deck/`. **Never remove or bypass this safety mechanism** — past incidents destroyed 36 production sessions and orphaned 20+ tmux sessions.

Use `skipIfNoTmuxServer(t)` for integration tests that need a running tmux server.

## Conventions

- Conventional commits: `feat:`, `fix:`, `chore:`, `refactor:`, `docs:` with optional `(scope)`
- Branch naming: `feature/`, `fix/`, `perf/`, `docs/`, `refactor/` prefixes
- Releases via GoReleaser, triggered by `v*` tags — builds for darwin/linux on amd64/arm64 with CGO disabled
