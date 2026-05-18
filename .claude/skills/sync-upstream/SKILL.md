---
name: sync-upstream
description: Sync the agent-deck local branch with upstream/main, skipping the main branch entirely. Use when the user says "sync upstream", "pull upstream", "update agent-deck from upstream", "/sync-upstream", or any variant. Performs a direct merge of upstream/main into local with conflict resolution and a post-merge build check.
---

# Sync agent-deck local branch with upstream

This is a personal fork (`c2keesey/agent-deck`) of `asheshgoplani/agent-deck`. The user works only on the `local` branch and does not maintain a separate `main` branch — sync `upstream/main` directly into `local`.

## Workflow

1. **Verify state** — Must be on `local` branch with a clean working tree. If dirty, stop and ask the user how to proceed (`git stash` vs. commit vs. abort). If on a different branch, ask before switching.

2. **Fetch and survey** — `git fetch upstream`. Show divergence: how many commits in `upstream/main..local` (your stuff) and `local..upstream/main` (new from upstream). If `local..upstream/main` is empty, report "already current" and stop.

3. **Merge** — `git merge upstream/main`. Do not rebase; the existing history pattern is `Merge branch 'main' into local` / similar.

4. **Resolve conflicts** — For each conflicted file:
   - Look at the conflict; usually upstream wins for shared code that has evolved, local wins for the user's customizations (e.g. `y` teardown hotkey, `teardownSession()` in `internal/ui/home.go`).
   - **CLAUDE.md**: upstream untracked it (PR #1002, file is already in `.gitignore` on local). Accept the deletion with `git rm --cached CLAUDE.md`; the on-disk file is preserved.
   - Watch for stale `case "X"` hotkey collisions in `internal/ui/home.go` (upstream remaps; local may still hold the old mapping).
   - After resolving, verify no markers remain: `grep -c "^<<<<<<\|^>>>>>>\|^======" <files>`.

5. **Build verification** — `GOTOOLCHAIN=go1.24.0 go build -o /tmp/agentdeck-sync-test ./cmd/agent-deck`. If it fails, fix imports / duplicates / stale references in the working tree BEFORE committing. Do not commit a broken merge.

6. **Commit** — Use a merge commit:
   ```
   Merge upstream/main into local

   Sync upstream v1.X.Y (~N commits) into local fork branch.

   Conflict resolutions: <one bullet per file>
   ```
   No `Co-Authored-By` line, no `--no-verify` on source changes (per CLAUDE.md mandates).

7. **Test gates (mention, do not auto-run)** — Per project CLAUDE.md, before pushing, the user should run:
   - `go test -run TestPersistence_ ./internal/session/... -race -count=1`
   - `go test ./internal/feedback/... ./internal/ui/... ./cmd/agent-deck/... -run "Feedback|Sender_" -race -count=1`
   - `go test ./internal/watcher/... -race -count=1 -timeout 120s`
   - Or `make ci` for the full suite.

8. **Do not push** — Per the user's global rule, never `git push` without explicit approval. Report the commit count ahead of `origin/local` and let the user push when ready.

## Notes

- `main` branch may be arbitrarily stale; that is intentional. Do not try to fast-forward or sync it.
- The deprecated `ad-sync` script and its launchd job were removed 2026-05-18. Do not reference them.
- If conflicts are deep (>5 files with >10 hunks each), pause and ask the user how they want to proceed rather than guessing.
