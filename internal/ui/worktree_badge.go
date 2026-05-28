package ui

import (
	"fmt"
	"math"
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

// goldenRatioConjugate is φ-1 ≈ 0.6180339887. Used as the increment in
// a low-discrepancy sequence over [0,1): successive points are placed
// maximally far from prior points (the same trick used by procedural
// graphics for color picking). Critically, this means the visual gap
// between the closest two hues shrinks slowly with N (~1/N²-ish) rather
// than collapsing immediately like a random hash distribution would.
const goldenRatioConjugate = 0.6180339887498949

// worktreeColors is the package-singleton color assigner. Hashing keys
// directly into a hue gave painful collisions (worker-1/3/5/7/9 all
// landed within 8° of each other under FNV-1a, even SHA-256 birthday-
// paradoxed worker-1/2/3/5 within 18° of each other). Insertion-order
// + golden-ratio mapping guarantees the closest two hues are at least
// ~360°/N² apart — visually distinct for any reasonable N.
//
// Trade-off: order matters. If sessions load in a different order
// across restarts (Go map iteration is randomized), colors will shuffle.
// Acceptable: the goal is "two sessions in the same worktree share a
// color in THIS view", not "worker-7 is always cyan forever."
var worktreeColors = newWorktreeColorAssigner()

type worktreeColorAssigner struct {
	mu      sync.Mutex
	colors  map[string]lipgloss.Color
	nextIdx int
}

func newWorktreeColorAssigner() *worktreeColorAssigner {
	return &worktreeColorAssigner{colors: map[string]lipgloss.Color{}}
}

// Color returns the assigned color for key, allocating one on first call.
// Cheap mutex on the hot render path is fine — this is called once per
// session row, ~dozens of times per render, not per-cell.
func (a *worktreeColorAssigner) Color(key string) lipgloss.Color {
	if key == "" {
		return lipgloss.Color("")
	}
	a.mu.Lock()
	defer a.mu.Unlock()
	if c, ok := a.colors[key]; ok {
		return c
	}
	// Place this key at the next point in the golden-ratio sequence.
	// Offset by 0.13 so the first key isn't pure red (hue=0) — that
	// reads as "error" against the existing status palette.
	pos := math.Mod(float64(a.nextIdx)*goldenRatioConjugate+0.13, 1.0)
	r, g, b := hslToRGB(pos, 0.62, 0.66)
	c := lipgloss.Color(fmt.Sprintf("#%02x%02x%02x", r, g, b))
	a.colors[key] = c
	a.nextIdx++
	return c
}

func worktreeColor(key string) lipgloss.Color {
	return worktreeColors.Color(key)
}

// hslToRGB converts HSL (each 0..1) to 8-bit RGB. Standard formula —
// kept inline to avoid a color library dependency.
func hslToRGB(h, s, l float64) (uint8, uint8, uint8) {
	if s == 0 {
		v := uint8(math.Round(l * 255))
		return v, v, v
	}
	var q float64
	if l < 0.5 {
		q = l * (1 + s)
	} else {
		q = l + s - l*s
	}
	p := 2*l - q
	r := hueToRGB(p, q, h+1.0/3.0)
	g := hueToRGB(p, q, h)
	b := hueToRGB(p, q, h-1.0/3.0)
	return uint8(math.Round(r * 255)), uint8(math.Round(g * 255)), uint8(math.Round(b * 255))
}

func hueToRGB(p, q, t float64) float64 {
	if t < 0 {
		t += 1
	}
	if t > 1 {
		t -= 1
	}
	switch {
	case t < 1.0/6.0:
		return p + (q-p)*6*t
	case t < 1.0/2.0:
		return q
	case t < 2.0/3.0:
		return p + (q-p)*(2.0/3.0-t)*6
	}
	return p
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
	style := lipgloss.NewStyle().Foreground(worktreeColor(key)).Bold(true)
	if selected {
		style = SessionStatusSelStyle.Bold(true)
	}
	return " " + style.Render(worktreeBadgeGlyph+" "+display)
}
