# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Fork Context

This is a personal fork (`c2keesey/agent-deck`) of `asheshgoplani/agent-deck`.

- **origin**: `c2keesey/agent-deck` (this fork)
- **upstream**: `asheshgoplani/agent-deck` (canonical repo)
- **`main` branch**: Mirrors upstream. Pull from upstream, push to origin.
- **`local` branch**: Personal customizations rebased/merged on top of main.
- **`ad-sync` script** (`~/.local/bin/ad-sync`): Pulls upstream into main, rebases local on top, rebuilds. Uses Claude headless to auto-resolve rebase conflicts via `/resolve-rebase`.
- **Scheduled daily at 9am** via launchd (`com.c2k.ad-sync`). Logs to `.ad-sync.log`. Skips if working tree is dirty.

When working on the `local` branch, keep changes minimal and mergeable — upstream moves fast.

## Build & Development

```bash
make build          # Build to ./build/agent-deck
make test           # Tests with -race detector
make lint           # golangci-lint
make fmt            # go fmt
make ci             # Full pre-push checks (lint + test + build via lefthook)
make dev            # Auto-reload (requires 'air')
```

Go toolchain is pinned to 1.24.0 (`GOTOOLCHAIN=go1.24.0` in Makefile) to avoid a Go 1.25 runtime regression on macOS.

### Running a single test

```bash
go test -race -run TestFunctionName ./internal/session/
go test -race -v -run TestSpecificCase ./internal/ui/
```

### Debug mode

```bash
AGENTDECK_DEBUG=1 agent-deck
```

## Architecture

Agent Deck is a terminal session manager for AI coding agents (Claude Code, Gemini, Codex, OpenCode, custom tools). It wraps tmux sessions with a Bubble Tea TUI and provides lifecycle management, cost tracking, and orchestration.

### Core data flow

```
User → TUI (internal/ui/) → Session Manager (internal/session/) → tmux (internal/tmux/)
                                    ↕                                      ↕
                              StateDB (SQLite)                    Pane output → Status detection
                              internal/statedb/                   internal/tmux/detector.go
```

### Key packages

| Package | Purpose |
|---------|---------|
| `cmd/agent-deck/` | CLI entry point, subcommand routing |
| `internal/session/` | Session lifecycle, config (userconfig.go), tool options, groups, conductor |
| `internal/ui/` | Bubble Tea TUI — `home.go` is the main view (~10K lines) |
| `internal/tmux/` | tmux abstraction, PTY, status detection via output pattern matching |
| `internal/statedb/` | SQLite persistence (sessions, groups, costs). WAL mode, multi-process safe |
| `internal/web/` | HTTP/WebSocket server + Preact frontend (port 8420) |
| `internal/mcppool/` | MCP socket pooling — shared Unix sockets across sessions (~85% memory reduction) |
| `internal/costs/` | Cost tracking with model-specific parsers (Claude, Gemini, OpenAI, MiniMax) |
| `internal/git/` | Worktree management, branch picking |
| `internal/docker/` | Docker sandbox support |

### Session lifecycle

1. **Create** → Instance struct in memory → persisted to statedb + JSON → tmux session started
2. **Detect** → Hook handlers or tmux polling find tool session IDs (e.g. `CLAUDE_SESSION_ID` from tmux env)
3. **Monitor** → Pattern matching on pane output determines status: `running`/`waiting`/`idle`/`error`/`stopped`
4. **Fork** → New Instance with same project; Claude sessions get conversation history via API fork

### Conductor system

Persistent meta-agent that monitors other sessions. Lives at `~/.agent-deck/conductor/<name>/` with its own `CLAUDE.md`, `meta.json`, and `state.json`. Heartbeat-driven polling, can escalate to user via Telegram/Slack bridges.

### Configuration

TOML-based at `~/.agent-deck/profiles/<profile>/config.toml`. Key sections: `[claude]`, `[gemini]`, `[codex]`, `[mcps]`, `[tools.*]`, `[worktree]`, `[docker]`, `[conductor]`, `[costs]`, `[notifications]`. Profile overrides via `[profiles.<name>.*]`.

### State persistence

```
~/.agent-deck/
├── profiles/<profile>/
│   ├── config.toml       # User config
│   ├── sessions.db       # SQLite (instances, groups, costs, transitions)
│   ├── sessions.json     # JSON backup
│   └── groups.json       # Group hierarchy
└── conductor/<name>/     # Conductor instances
```

## Testing conventions

- Tests live alongside source in `*_test.go` files
- Integration tests use `_integration_test.go` suffix
- Tests that need tmux skip via `skipIfNoTmuxServer(t)`
- `TestMain()` in `testmain_test.go` unsets `GIT_DIR`/`GIT_WORK_TREE` to prevent interference
- Shared git test helpers in `internal/testutil/gitenv.go`
- Race detector is always on: `go test -race`

## Commit conventions

Conventional commits: `feat:`, `fix:`, `perf:`, `docs:`, `refactor:`. Branch naming: `feature/`, `fix/`, `perf/`, `docs/`, `refactor/`.
