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

## Session persistence: mandatory test coverage

Agent-deck has a recurring production failure where a single SSH logout on a Linux+systemd host destroys **every** managed tmux session. **As of v1.5.2, this class of bug is permanently test-gated.**

### The eight required tests

Any PR modifying session lifecycle paths MUST run `go test -run TestPersistence_ ./internal/session/... -race -count=1`. In addition, `bash scripts/verify-session-persistence.sh` MUST run end-to-end on a Linux+systemd host.

### Paths under the mandate

- `internal/tmux/**`, `internal/session/instance.go`, `internal/session/userconfig.go`, `internal/session/storage*.go`
- `cmd/session_cmd.go`, `scripts/verify-session-persistence.sh`, this `CLAUDE.md` section

### Forbidden changes without an RFC

- Flipping `launch_in_user_scope` default back to `false` on Linux
- Removing any of the eight `TestPersistence_*` tests
- Adding a code path that starts a Claude session and ignores `Instance.ClaudeSessionID`

## Feedback feature: mandatory test coverage

The in-product feedback feature is covered by 23 tests. All must pass before any PR touching the feedback surface is merged.

```
go test ./internal/feedback/... ./internal/ui/... ./cmd/agent-deck/... -run "Feedback|Sender_" -race -count=1
```

Reintroducing `D_PLACEHOLDER` as `feedback.DiscussionNodeID` is a **blocker**. `TestSender_DiscussionNodeID_IsReal` catches this automatically.

## Per-group config: mandatory test coverage

Per-group config dir applies to custom-command sessions too; `TestPerGroupConfig_*` suite enforces this.

## Watcher framework: mandatory test coverage

Any commit touching watcher source code MUST pass:

```bash
go test ./internal/watcher/... -race -count=1 -timeout 120s
go test ./cmd/agent-deck/... -run "Watcher" -race -count=1
```

### Watcher paths under the mandate

- `internal/watcher/**` (engine, adapters, health bridge, layout, state, event log, router)
- `cmd/agent-deck/watcher_cmd*.go` (CLI surface)
- `internal/ui/watcher_panel.go` (TUI watcher panel)
- `internal/statedb/statedb.go` (watcher rows in SQLite)
- `cmd/agent-deck/assets/skills/watcher-creator/` (embedded skill)
- `internal/session/watcher_meta.go` (watcher directory helpers)

### Watcher structural changes requiring RFC

- Removing or weakening the health bridge (`internal/watcher/health_bridge.go`)
- Disabling SQLite dedup (INSERT OR IGNORE on `watcher_events`)
- Weakening HMAC-SHA256 verification on the GitHub adapter
- Changing the `~/.agent-deck/watcher/` folder layout (REQ-WF-6)

### Skills + docs sync (REQ-WF-7)

Any commit modifying `internal/watcher/layout.go` or `internal/session/watcher_meta.go` MUST also update embedded skills, README, and CHANGELOG. `TestSkillDriftCheck_WatcherCreator` enforces this at build time.

### Integration harness

```bash
bash scripts/verify-watcher-framework.sh
```

## --no-verify mandate

**`git commit --no-verify` is FORBIDDEN on source-modifying commits.** Metadata-only commits (`.planning/**`, `docs/**`, non-source `*.md`) MAY use `--no-verify` when hooks would no-op.

## General rules

- **Never `rm`** — use `trash`.
- **Never commit with Claude attribution** — no "Generated with Claude Code" or "Co-Authored-By: Claude" lines.
- **Never `git push`, `git tag`, `gh release`, `gh pr create/merge`** without explicit user approval.
- **TDD always** — the regression test for a bug lands BEFORE the fix.
- **Simplicity first** — every change minimal, targeted, no speculative refactoring.
