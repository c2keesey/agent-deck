---
name: sync-upstream
description: Sync the agent-deck local branch with upstream/main, skipping the main branch entirely. Use when the user says "sync upstream", "pull upstream", "update agent-deck from upstream", "/sync-upstream", or any variant. Performs a direct merge of upstream/main into local with conflict resolution, an auto-run test gate, a feature summary, and an auto-push.
---

# Sync agent-deck local branch with upstream

This is a personal fork (`c2keesey/agent-deck`) of `asheshgoplani/agent-deck`. The user works only on the `local` branch and does not maintain a separate `main` branch — sync `upstream/main` directly into `local`.

## Workflow

1. **Verify state** — Must be on `local` branch with a clean working tree. If dirty, stop and ask the user how to proceed (`git stash` vs. commit vs. abort). If on a different branch, ask before switching.

2. **Fetch and survey** — `git fetch upstream`. Show divergence: how many commits in `upstream/main..local` (your stuff) and `local..upstream/main` (new from upstream). If `local..upstream/main` is empty, report "already current" and stop.

3. **Merge** — `git merge upstream/main`. Do not rebase; the existing history pattern is `Merge branch 'main' into local` / similar.

4. **Resolve conflicts** — For each conflicted file:
   - **ALWAYS PREFER LOCAL FEATURES.** On any conflict where local has a deliberate customization, take the local side — even when upstream ships its own "official" version of a similar feature. Only take upstream for shared code that has genuinely evolved and where local has no intentional divergence. (Memory: `prefer-local-over-upstream`.)
   - If an upstream test enforces upstream's behavior and contradicts a kept local feature, **rewrite that test to pin the local behavior** (don't just delete coverage) and add a comment noting the divergence.
   - **CLAUDE.md**: upstream untracked it (PR #1002, file is already in `.gitignore` on local). Accept the deletion with `git rm --cached CLAUDE.md`; the on-disk file is preserved.
   - Watch for stale `case "X"` hotkey collisions in `internal/ui/home.go` (upstream remaps; local may still hold the old mapping).
   - After resolving, verify no markers remain: `grep -rn "^<<<<<<<\|^>>>>>>>\|^=======" --include='*.go' .`

   **Known fork features that collide with upstream — watch these areas:**
   - `y` teardown hotkey + `teardownSession()` in `internal/ui/home.go`.
   - **Always-broadcast pane-title slot + status fallback** (row render in `home.go`) vs upstream's `show_pane_titles` toggle (#1343). Keep local always-on.
   - **Personal-fork curated footer** (right-pads to width, shows `n/a new`) vs upstream's opt-in `[ui] footer` curated mode (#1300/#1289). The footer right-pad makes substring tests like `LastIndex("p ")` false-match inside "hel**p** " — anchor footer-order tests on label tokens (`"p settings"`, `"? help"`), not `key+" "`.
   - **Title-lock on rename** (`SetField(FieldTitle)` sets `TitleLocked`) — upstream #1355 is the canonical version; adopt upstream here.
   - **Start-path / restart session-id binding** (`ensureClaudeSessionIDFromDisk*` in `instance.go`) — adjacent to upstream #1324 (codex history scan) and #1349 (rebind guard). Re-run `TestStartPath_*` and `TestIssue1147_*` after.
   - **MAIA new-session picker** (codex always-YOLO, raw shell, `~/home` claude) — overlaps upstream new-session dialog nav (#1295), tool-visibility denylist (#1346), shell quick-create (#1307).
   - **happy-wrapper fork support** — local has RED TDD stubs (`*UseHappy*` tests) for a feature not yet implemented; these fail pre- and post-merge. Do NOT treat as merge regressions.

5. **Build verification** — `go build -o /tmp/agentdeck-sync-test ./cmd/agent-deck`. (Do NOT pin `GOTOOLCHAIN=go1.24.0` — go.mod tracks upstream's required version, currently ≥1.25.11.) If it fails, fix imports / duplicates / stale references in the working tree BEFORE committing. Do not commit a broken merge.

6. **Commit** — Use a merge commit:
   ```
   Merge upstream/main into local

   Sync upstream v1.X.Y (~N commits) into local fork branch.

   Conflict resolutions: <one bullet per file>
   <one line on pre-existing vs merge-regression failures>
   ```
   No `Co-Authored-By` line, no `--no-verify` on source changes (per CLAUDE.md mandates). Pre-commit runs gofmt + vet only (fast); the full suite is pre-push.

7. **Run tests (auto-run) and classify failures.** Run the mandated gates plus the full suite:
   ```
   go test -run TestPersistence_ ./internal/session/... -race -count=1
   go test ./internal/feedback/... ./internal/ui/... ./cmd/agent-deck/... -run "Feedback|Sender_" -race -count=1
   go test ./internal/watcher/... -race -count=1 -timeout 120s
   go test ./... -count=1     # broad sweep to catch merge regressions
   ```
   For EACH failing test, classify it before deciding whether it blocks the push:
   - **Pre-existing local failure** — also fails on the pre-merge commit. Check with a throwaway worktree: `git worktree add -q /tmp/ad-premerge HEAD^1 && (cd /tmp/ad-premerge && go test -run <Test> ...)`. Not a blocker.
   - **Environmental / upstream-side failure** — also fails on clean `upstream/main`: `git worktree add -q /tmp/ad-upstream upstream/main && (cd /tmp/ad-upstream && go test -run <Test> ...)`. Not a blocker. (Common: macOS `/var/folders`→`/private/var/folders` symlink tests like `GetJSONLPathChecked_*`, `*CustomCommandResumes*`; and the `*UseHappy*` local stubs.)
   - **Merge regression** — passes on BOTH parents but fails on the merge. THIS BLOCKS. Fix it before pushing; if you can't, stop and report.
   - Clean up: `git worktree remove /tmp/ad-premerge; git worktree remove /tmp/ad-upstream; git worktree prune`. (Never use `--force` — Safety Net blocks it.)

8. **Feature summary** — Produce a summary of what landed, grouped: (a) features/fixes that touch or overlap the user's fork customizations (from the list in step 4) — call these out first and explicitly, (b) other notable user-facing features, (c) infra/CI/deps in one line. Source it from `git log --no-merges --pretty='%s' <newtag> --not <pre-merge-commit>^ | grep -iE '^(feat|fix|perf)'`.

9. **Push (auto).** The user is pre-authorized to push the `local` branch to origin (memory: `push-local-freely`). After step 7 shows no merge regressions, `git push origin local` automatically. Report the commit count pushed and the new `origin/local` position. If there IS a merge regression, do NOT push — report and stop.

## Notes

- `main` branch may be arbitrarily stale; that is intentional. Do not try to fast-forward or sync it.
- The deprecated `ad-sync` script and its launchd job were removed 2026-05-18. Do not reference them.
- If conflicts are deep (>5 files with >10 hunks each), pause and ask the user how they want to proceed rather than guessing.
