package ui

import (
	"hash/fnv"
	"path/filepath"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/asheshgoplani/agent-deck/internal/session"
)

// worktreePalette is the set of ANSI 256 colors used to visually group
// sessions that share a worktree. Picked for high contrast against the
// default dark/light themes and distinct from the status colors (green
// for running, yellow for waiting, red for error). Same worktreeKey
// always hashes to the same index, so all sessions in MAIA.worker-7 get
// the same color across renders.
var worktreePalette = []lipgloss.Color{
	lipgloss.Color("33"),  // blue
	lipgloss.Color("165"), // magenta
	lipgloss.Color("51"),  // cyan
	lipgloss.Color("208"), // orange
	lipgloss.Color("141"), // purple
	lipgloss.Color("113"), // mint green
	lipgloss.Color("167"), // pink
	lipgloss.Color("75"),  // sky
	lipgloss.Color("178"), // gold
	lipgloss.Color("99"),  // violet
	lipgloss.Color("117"), // pale blue
	lipgloss.Color("180"), // tan
}

// worktreeKey returns the display identifier for a session's worktree —
// the thing that decides both the badge text and the color bucket.
// Priority:
//  1. WorktreeBranch (real git worktree created via agent-deck)
//  2. ProjectPath basename (pre-created worktrees like MAIA.worker-N,
//     where ProjectPath itself IS the worktree dir and WorktreeBranch
//     is empty)
//  3. "" when neither is available — render no badge.
func worktreeKey(inst *session.Instance) string {
	if inst == nil {
		return ""
	}
	if inst.WorktreeBranch != "" {
		return strings.TrimPrefix(inst.WorktreeBranch, "MAIA.")
	}
	if inst.ProjectPath == "" {
		return ""
	}
	base := filepath.Base(strings.TrimRight(inst.ProjectPath, "/"))
	if base == "" || base == "." || base == "/" {
		return ""
	}
	// Normalize the MAIA. prefix out of the key so both the badge text
	// and the color bucket are stable on the bare worker identifier.
	return strings.TrimPrefix(base, "MAIA.")
}

// worktreeColor maps a key to a stable color from worktreePalette via
// FNV-1a. Sessions sharing a key always get the same color.
func worktreeColor(key string) lipgloss.Color {
	if key == "" {
		return lipgloss.Color("")
	}
	h := fnv.New32a()
	_, _ = h.Write([]byte(key))
	return worktreePalette[int(h.Sum32())%len(worktreePalette)]
}

// renderWorktreeBadge produces the colored "[worktree-name]" tag shown
// next to each session in the dashboard. Truncates long keys so the row
// stays bounded.
func renderWorktreeBadge(inst *session.Instance, selected bool) string {
	key := worktreeKey(inst)
	if key == "" {
		return ""
	}
	display := key
	if len(display) > 20 {
		display = display[:17] + "..."
	}
	style := lipgloss.NewStyle().Foreground(worktreeColor(key))
	if selected {
		style = SessionStatusSelStyle
	}
	return style.Render(" [" + display + "]")
}
