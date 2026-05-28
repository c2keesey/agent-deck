package ui

import (
	"path/filepath"
	"strings"
	"sync"

	"github.com/charmbracelet/lipgloss"

	"github.com/asheshgoplani/agent-deck/internal/session"
)

// worktreeKey returns the display identifier for a session's worktree —
// the thing that decides both the badge text and the color bucket.
// Priority:
//  1. WorktreeBranch (real git worktree created via agent-deck)
//  2. ProjectPath basename (pre-created worktrees like MAIA.worker-N,
//     where ProjectPath itself IS the worktree dir and WorktreeBranch
//     is empty)
//  3. "" when neither is available — render no badge.
//
// The "MAIA." prefix is stripped from both inputs so the bare worker
// identifier (e.g. "worker-7", "ro-dev") drives the key.
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
	return strings.TrimPrefix(base, "MAIA.")
}

// worktreePalette is a hand-picked set of 24 ANSI 256 colors chosen to
// be (a) visually distinct from each other, (b) distinct from the
// status colors (green/yellow/red), and (c) legible on both dark and
// light themes. ANSI 256 codes are used instead of hex so termenv
// renders them identically across every terminal profile — switching to
// truecolor hex strings was the previous attempt and rendered as black
// somewhere in the lipgloss/termenv pipeline despite the 24-bit profile
// being active. ANSI is the safe choice.
//
// Ordering matters: keys are assigned colors by insertion index into a
// golden-ratio sequence (see worktreeColorAssigner), so adjacent
// indices map to maximally-distant palette slots.
var worktreePalette = []lipgloss.Color{
	lipgloss.Color("33"),  // blue
	lipgloss.Color("165"), // magenta
	lipgloss.Color("51"),  // cyan
	lipgloss.Color("208"), // orange
	lipgloss.Color("141"), // purple
	lipgloss.Color("113"), // mint
	lipgloss.Color("167"), // salmon
	lipgloss.Color("75"),  // sky
	lipgloss.Color("178"), // gold
	lipgloss.Color("99"),  // violet
	lipgloss.Color("44"),  // teal
	lipgloss.Color("203"), // coral
	lipgloss.Color("69"),  // periwinkle
	lipgloss.Color("213"), // pink
	lipgloss.Color("39"),  // deep sky
	lipgloss.Color("215"), // peach
	lipgloss.Color("129"), // plum
	lipgloss.Color("85"),  // jade
	lipgloss.Color("173"), // tan-rose
	lipgloss.Color("105"), // lavender
	lipgloss.Color("80"),  // turquoise
	lipgloss.Color("219"), // bubblegum
	lipgloss.Color("63"),  // indigo
	lipgloss.Color("221"), // sand
}

// worktreeColors is the package-singleton color assigner. We deliberately
// avoid hashing the key directly into a palette index — that birthday-
// paradoxed worker-1/3/5/7/9 into the same bucket on the user's actual
// key set. Insertion-order with a golden-ratio step (chosen so successive
// indices land far apart in the palette) keeps the closest two colors as
// far apart as possible for any N up to len(palette).
var worktreeColors = newWorktreeColorAssigner()

type worktreeColorAssigner struct {
	mu      sync.Mutex
	colors  map[string]lipgloss.Color
	nextIdx int
}

func newWorktreeColorAssigner() *worktreeColorAssigner {
	return &worktreeColorAssigner{colors: map[string]lipgloss.Color{}}
}

// goldenStep is chosen so successive insertion indices map to maximally
// distant palette slots. It must be coprime to len(worktreePalette)
// (24) so the sequence visits every slot before repeating. 13 satisfies
// both: gcd(13,24)=1, and 13/24 ≈ 0.54 is close to the golden-ratio
// conjugate. The first 14 assignments visit slots 0, 13, 2, 15, 4, 17,
// 6, 19, 8, 21, 10, 23, 12, 1 — every other side of the palette.
const goldenStep = 13

// Color returns the assigned color for key, allocating one on first call.
// Cheap mutex on the hot render path is fine — called once per session
// row, ~dozens of times per render at most.
func (a *worktreeColorAssigner) Color(key string) lipgloss.Color {
	if key == "" {
		return lipgloss.Color("")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if c, ok := a.colors[key]; ok {
		return c
	}
	idx := (a.nextIdx * goldenStep) % len(worktreePalette)
	c := worktreePalette[idx]
	a.colors[key] = c
	a.nextIdx++
	return c
}

func worktreeColor(key string) lipgloss.Color {
	return worktreeColors.Color(key)
}

// worktreeBadgeGlyph is the visual anchor that precedes the worktree
// name. ⎇ (U+2387, "alternative key") is widely used in git/branch
// contexts (vim airline, starship prompts), reads as "branch" at a
// glance, and renders as a single cell in standard monospace fonts.
const worktreeBadgeGlyph = "⎇"

// renderWorktreeBadge renders the worktree tag for a session row. The
// color is the load-bearing identity signal; the glyph is a small,
// consistent visual anchor so the eye finds the same column across rows
// without the brackets that previously framed it.
func renderWorktreeBadge(inst *session.Instance, selected bool) string {
	key := worktreeKey(inst)
	if key == "" {
		return ""
	}
	display := key
	if len(display) > 20 {
		display = display[:17] + "…"
	}
	style := lipgloss.NewStyle().Foreground(worktreeColor(key))
	if selected {
		style = SessionStatusSelStyle
	}
	return " " + style.Render(worktreeBadgeGlyph+" "+display)
}
