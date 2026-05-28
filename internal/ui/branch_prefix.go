package ui

import (
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// branchCache memoizes git HEAD lookups per repo dir. Branches change
// rarely (every few minutes at most during active dev), so a 30s TTL
// avoids re-reading .git/HEAD on every status snapshot tick while still
// picking up branch switches within a render cycle.
var (
	branchCacheMu sync.RWMutex
	branchCache   = map[string]branchCacheEntry{}
)

type branchCacheEntry struct {
	branch string
	at     time.Time
}

const branchCacheTTL = 30 * time.Second

// cachedGitBranch returns the current branch for the worktree rooted at
// repoDir, falling back to "" when the dir isn't a git checkout or HEAD
// is detached/unreadable. Cached behind a mutex with a short TTL — fast
// enough for the render-snapshot path (called once per session per
// background status tick).
func cachedGitBranch(repoDir string) string {
	if repoDir == "" {
		return ""
	}
	branchCacheMu.RLock()
	e, ok := branchCache[repoDir]
	branchCacheMu.RUnlock()
	if ok && time.Since(e.at) < branchCacheTTL {
		return e.branch
	}
	b := readGitBranch(repoDir)
	branchCacheMu.Lock()
	branchCache[repoDir] = branchCacheEntry{branch: b, at: time.Now()}
	branchCacheMu.Unlock()
	return b
}

// readGitBranch resolves the current HEAD ref by reading .git/HEAD
// directly. Handles three cases:
//
//   - .git is a directory → standard repo, read .git/HEAD.
//   - .git is a file → linked worktree, .git contains "gitdir: <path>"
//     pointing at the per-worktree HEAD inside the main repo.
//   - HEAD points at a ref (`ref: refs/heads/branch`) → return the
//     short branch name. Detached HEAD (a raw SHA) → return short SHA.
//
// Reading files instead of shelling out to `git branch --show-current`
// is ~100x faster and avoids forking on every status tick. The cost is
// that this won't handle exotic git configurations (worktree of a
// worktree, packed refs in the HEAD itself) — but for the standard
// `git worktree add` layout the user has, that's fine.
func readGitBranch(repoDir string) string {
	gitPath := filepath.Join(repoDir, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return ""
	}
	var headPath string
	if info.IsDir() {
		headPath = filepath.Join(gitPath, "HEAD")
	} else {
		data, err := os.ReadFile(gitPath)
		if err != nil {
			return ""
		}
		line := strings.TrimSpace(string(data))
		if !strings.HasPrefix(line, "gitdir:") {
			return ""
		}
		gitdir := strings.TrimSpace(strings.TrimPrefix(line, "gitdir:"))
		// gitdir paths are usually absolute, but accept relative-to-worktree too.
		if !filepath.IsAbs(gitdir) {
			gitdir = filepath.Join(repoDir, gitdir)
		}
		headPath = filepath.Join(gitdir, "HEAD")
	}
	data, err := os.ReadFile(headPath)
	if err != nil {
		return ""
	}
	line := strings.TrimSpace(string(data))
	if strings.HasPrefix(line, "ref:") {
		ref := strings.TrimSpace(strings.TrimPrefix(line, "ref:"))
		return strings.TrimPrefix(ref, "refs/heads/")
	}
	// Detached HEAD — return short SHA so the user has SOME identifier.
	if len(line) >= 7 {
		return line[:7]
	}
	return ""
}

// maxBranchDisplayCells caps the branch prefix at a reasonable width.
// Long ticket-prefixed branches (e.g. "ck/MAIA-1863-register-layer-sql-
// deterministic") would otherwise push the worktree badge off the row;
// truncating to ~24 cells keeps the ticket identifier visible.
const maxBranchDisplayCells = 24

// renderBranchPrefix renders the dim branch tag that sits before the
// session title. Empty string when there's no branch to show (.git not
// found, detached HEAD with no SHA, etc.) — the row format string drops
// it naturally.
func renderBranchPrefix(branch string, selected bool) string {
	if branch == "" {
		return ""
	}
	display := branch
	if cellWidth(display) > maxBranchDisplayCells {
		display = cellTruncate(display, maxBranchDisplayCells, "…")
	}
	style := lipgloss.NewStyle().Foreground(ColorTextDim)
	if selected {
		style = SessionStatusSelStyle
	}
	return " " + style.Render(display)
}
