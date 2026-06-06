package ui

import (
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// maiaReposDir is the parent directory scanned for MAIA worktrees.
// Personal fork customization.
const maiaReposDir = "/Users/c2k/MAIA/Repos"

// Groups the picker pins new sessions to, per the user's workflow:
// workers land in maia/active, read-only dev sessions in maia/read-only-dev.
const (
	maiaWorkerGroup = "maia/active"
	maiaRoDevGroup  = "maia/read-only-dev"
)

// MaiaWorkerPicker is the two-column new-session picker for MAIA work.
//
//   - Left column: worker worktrees (MAIA.worker-*). The cursor defaults to
//     the next OPEN worker — the lowest-numbered worktree with no agent-deck
//     session in it (occupied = any session, regardless of status).
//   - Right column: ro-dev worktrees (MAIA.ro-dev*) for read-only dev
//     sessions. Enter just starts another session in the shared ro-dev
//     worktree (no branch/worktree creation).
//
// ←/→ (or h/l) switch columns, ↑/↓ (or j/k) move within a column, Enter
// creates a Claude session in the selected worktree with the column's group.
type MaiaWorkerPicker struct {
	visible      bool
	workers      []string        // worker worktree paths
	roDevs       []string        // ro-dev worktree paths
	occupied     map[string]bool // worktree path -> already hosts a session
	focusCol     int             // 0 = workers, 1 = ro-dev
	workerCursor int
	roDevCursor  int
	width        int
	height       int
	scanErr      string
}

// NewMaiaWorkerPicker constructs an empty picker; worktrees are scanned on
// each Show() so freshly-created worktrees appear without a restart.
func NewMaiaWorkerPicker() *MaiaWorkerPicker { return &MaiaWorkerPicker{} }

// Show opens the picker. occupied maps worktree paths that already host an
// agent-deck session (any status); the worker cursor parks on the first
// worktree not in that set.
func (m *MaiaWorkerPicker) Show(occupied map[string]bool) {
	m.visible = true
	m.occupied = occupied
	m.focusCol = 0
	m.roDevCursor = 0
	m.refreshWorktrees()
	m.workerCursor = m.nextOpenWorker()
}

// Hide closes the picker.
func (m *MaiaWorkerPicker) Hide() { m.visible = false }

// IsVisible reports whether the picker is shown. Nil-safe so the key-router
// and View overlay checks don't panic on a Home built without a picker.
func (m *MaiaWorkerPicker) IsVisible() bool { return m != nil && m.visible }

// SetSize updates the viewport for centering.
func (m *MaiaWorkerPicker) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Selected returns the worktree path under the cursor in the focused column
// and the group its session should join. Empty path means nothing to create.
func (m *MaiaWorkerPicker) Selected() (path, group string) {
	if m.focusCol == 1 {
		if m.roDevCursor >= 0 && m.roDevCursor < len(m.roDevs) {
			return m.roDevs[m.roDevCursor], maiaRoDevGroup
		}
		return "", ""
	}
	if m.workerCursor >= 0 && m.workerCursor < len(m.workers) {
		return m.workers[m.workerCursor], maiaWorkerGroup
	}
	return "", ""
}

// Update handles navigation keys (Enter/Esc are handled by the caller).
func (m *MaiaWorkerPicker) Update(msg tea.KeyMsg) (*MaiaWorkerPicker, tea.Cmd) {
	switch msg.String() {
	case "up", "ctrl+p", "k":
		if m.focusCol == 0 {
			if m.workerCursor > 0 {
				m.workerCursor--
			}
		} else if m.roDevCursor > 0 {
			m.roDevCursor--
		}
	case "down", "ctrl+n", "j":
		if m.focusCol == 0 {
			if m.workerCursor < len(m.workers)-1 {
				m.workerCursor++
			}
		} else if m.roDevCursor < len(m.roDevs)-1 {
			m.roDevCursor++
		}
	case "right", "l", "tab":
		if len(m.roDevs) > 0 {
			m.focusCol = 1
		}
	case "left", "h", "shift+tab":
		m.focusCol = 0
	}
	return m, nil
}

// refreshWorktrees scans maiaReposDir, splitting MAIA.worker-* into the left
// column and MAIA.ro-dev* into the right. Workers sort numerically
// (worker-2 < worker-10); non-numeric suffixes (worker-retry) sort last.
func (m *MaiaWorkerPicker) refreshWorktrees() {
	m.workers = nil
	m.roDevs = nil
	m.scanErr = ""
	entries, err := os.ReadDir(maiaReposDir)
	if err != nil {
		m.scanErr = err.Error()
		return
	}
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		name := e.Name()
		path := filepath.Join(maiaReposDir, name)
		switch {
		case strings.HasPrefix(name, "MAIA.worker-"):
			m.workers = append(m.workers, path)
		case name == "MAIA.ro-dev" || strings.HasPrefix(name, "MAIA.ro-dev"):
			m.roDevs = append(m.roDevs, path)
		}
	}
	sort.SliceStable(m.workers, func(i, j int) bool {
		return workerSortKey(m.workers[i]) < workerSortKey(m.workers[j])
	})
	sort.Strings(m.roDevs)
	if m.workerCursor >= len(m.workers) {
		m.workerCursor = 0
	}
	if m.roDevCursor >= len(m.roDevs) {
		m.roDevCursor = 0
	}
}

// workerSortKey extracts the numeric worker index for ordering; non-numeric
// suffixes (e.g. "worker-retry") sort last.
func workerSortKey(path string) int {
	suffix := strings.TrimPrefix(filepath.Base(path), "MAIA.worker-")
	if n, err := strconv.Atoi(suffix); err == nil {
		return n
	}
	return 1 << 30
}

// nextOpenWorker returns the index of the first worker worktree with no
// session. Falls back to 0 when all are occupied (or there are none).
func (m *MaiaWorkerPicker) nextOpenWorker() int {
	for i, p := range m.workers {
		if !m.occupied[p] {
			return i
		}
	}
	return 0
}

// View renders the two columns side by side, centered.
func (m *MaiaWorkerPicker) View() string {
	if !m.visible {
		return ""
	}

	title := DialogTitleStyle.Render("New MAIA Session")

	var body string
	if m.scanErr != "" {
		body = lipgloss.NewStyle().Foreground(ColorRed).Render("⚠ " + m.scanErr)
	} else {
		left := m.renderColumn("Workers", m.workers, m.workerCursor, m.focusCol == 0, true)
		right := m.renderColumn("ro-dev", m.roDevs, m.roDevCursor, m.focusCol == 1, false)
		body = lipgloss.JoinHorizontal(lipgloss.Top, left, "  ", right)
	}

	hint := lipgloss.NewStyle().Foreground(ColorComment).
		Render("↑/↓ pick · ←/→ column · Enter create · c codex · s shell · ~ home · Esc cancel")

	content := lipgloss.JoinVertical(lipgloss.Left, title, "", body, "", hint)
	dialog := DialogBoxStyle.Render(content)
	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, dialog)
}

// renderColumn renders one bordered column. The focused column gets an
// accent border and a highlighted cursor row; markOpen tags worker rows
// with an open (●) / busy (·) dot.
func (m *MaiaWorkerPicker) renderColumn(title string, items []string, cursor int, focused, markOpen bool) string {
	const colWidth = 22

	headerStyle := lipgloss.NewStyle().Foreground(ColorTextDim).Bold(true)
	rowStyle := lipgloss.NewStyle().Foreground(ColorText)
	selStyle := lipgloss.NewStyle().Foreground(ColorBg).Background(ColorAccent).Bold(true)
	openStyle := lipgloss.NewStyle().Foreground(ColorGreen)
	busyStyle := lipgloss.NewStyle().Foreground(ColorTextDim)

	var b strings.Builder
	b.WriteString(headerStyle.Render(title))
	b.WriteString("\n")

	if len(items) == 0 {
		b.WriteString(busyStyle.Render("(none found)"))
	}
	for i, p := range items {
		label := strings.TrimPrefix(filepath.Base(p), "MAIA.")
		marker := ""
		if markOpen {
			if m.occupied[p] {
				marker = busyStyle.Render(" ·")
			} else {
				marker = openStyle.Render(" ●")
			}
		}
		if focused && i == cursor {
			b.WriteString(selStyle.Render(" " + label + " "))
		} else {
			b.WriteString(rowStyle.Render("  " + label))
		}
		b.WriteString(marker)
		if i < len(items)-1 {
			b.WriteString("\n")
		}
	}

	box := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		Width(colWidth).
		Padding(0, 1)
	if focused {
		box = box.BorderForeground(ColorAccent)
	} else {
		box = box.BorderForeground(ColorBorder)
	}
	return box.Render(b.String())
}
