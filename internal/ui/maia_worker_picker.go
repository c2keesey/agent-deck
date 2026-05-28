package ui

import (
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/asheshgoplani/agent-deck/internal/session"
)

// maiaReposDir is the parent directory scanned for MAIA worktrees.
// Each entry matching the MAIA. prefix (plus the bare "MAIA" root) is
// listed in the simple new-session picker. Personal fork customization.
const maiaReposDir = "/Users/c2k/MAIA/Repos"

// MaiaWorkerPicker is the minimal new-session picker that replaces the
// full NewDialog flow for MAIA work: pick a worktree directory, everything
// else (tool=claude, group=maia/active, auto name) is defaulted.
//
// A textinput sits above the list. When empty, Enter creates a session in
// the highlighted MAIA worktree (group=maia/active). When non-empty, Enter
// creates a session in the typed path with whatever group quickCreateSessionAt
// derives — the escape hatch for non-MAIA dirs.
type MaiaWorkerPicker struct {
	visible    bool
	pathInput  textinput.Model
	worktrees  []string
	cursor     int
	width      int
	height     int
	scanErr    string
}

// NewMaiaWorkerPicker constructs a picker. Worktrees are scanned on each
// Show() so newly-created worktrees show up without restarting agent-deck.
func NewMaiaWorkerPicker() *MaiaWorkerPicker {
	ti := textinput.New()
	ti.Placeholder = "(or type a custom path — for non-MAIA dirs)"
	ti.CharLimit = 1024
	ti.Width = 50
	return &MaiaWorkerPicker{pathInput: ti}
}

// Show opens the picker and refreshes the worktree list.
func (m *MaiaWorkerPicker) Show() {
	m.visible = true
	m.cursor = 0
	m.pathInput.SetValue("")
	m.pathInput.CursorEnd()
	m.pathInput.Focus()
	m.refreshWorktrees()
}

// Hide closes the picker.
func (m *MaiaWorkerPicker) Hide() {
	m.visible = false
	m.pathInput.Blur()
}

// IsVisible reports whether the picker is currently shown.
func (m *MaiaWorkerPicker) IsVisible() bool { return m.visible }

// SelectedPath returns the path that Enter should act on:
//   - if the textinput is non-empty, that typed path (custom dir mode)
//   - otherwise the highlighted MAIA worktree
//   - empty if neither is selectable
func (m *MaiaWorkerPicker) SelectedPath() string {
	if typed := strings.TrimSpace(m.pathInput.Value()); typed != "" {
		return session.ExpandPath(typed)
	}
	if m.cursor < 0 || m.cursor >= len(m.worktrees) {
		return ""
	}
	return m.worktrees[m.cursor]
}

// IsCustomPath reports whether the user typed a custom path (vs. picked
// from the MAIA list). Callers use this to decide on the default group:
// MAIA pick → maia/active, custom path → derive from path.
func (m *MaiaWorkerPicker) IsCustomPath() bool {
	return strings.TrimSpace(m.pathInput.Value()) != ""
}

// SetSize updates the dialog viewport for centering.
func (m *MaiaWorkerPicker) SetSize(width, height int) {
	m.width = width
	m.height = height
}

// Update processes a key event. Arrow keys navigate the MAIA list even
// while the textinput has focus, so the user can type a path AND still
// scroll the list without an explicit focus switch.
func (m *MaiaWorkerPicker) Update(msg tea.KeyMsg) (*MaiaWorkerPicker, tea.Cmd) {
	switch msg.String() {
	case "up", "ctrl+p":
		if m.cursor > 0 {
			m.cursor--
		}
		return m, nil
	case "down", "ctrl+n":
		if m.cursor < len(m.worktrees)-1 {
			m.cursor++
		}
		return m, nil
	}
	var cmd tea.Cmd
	m.pathInput, cmd = m.pathInput.Update(msg)
	return m, cmd
}

// refreshWorktrees scans maiaReposDir for directories named "MAIA" or
// "MAIA.*". Entries are sorted alphabetically; the bare "MAIA" root sorts
// to the top of the .* group naturally.
func (m *MaiaWorkerPicker) refreshWorktrees() {
	m.worktrees = nil
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
		if name != "MAIA" && !strings.HasPrefix(name, "MAIA.") {
			continue
		}
		m.worktrees = append(m.worktrees, filepath.Join(maiaReposDir, name))
	}
	sort.Strings(m.worktrees)
	if m.cursor >= len(m.worktrees) {
		m.cursor = 0
	}
}

// View renders the overlay, centered.
func (m *MaiaWorkerPicker) View() string {
	if !m.visible {
		return ""
	}

	title := DialogTitleStyle.Render("New MAIA Session")

	var listBlock string
	switch {
	case m.scanErr != "":
		listBlock = lipgloss.NewStyle().
			Foreground(ColorRed).
			Render("⚠ " + m.scanErr)
	case len(m.worktrees) == 0:
		listBlock = lipgloss.NewStyle().
			Foreground(ColorTextDim).
			Render("(no MAIA worktrees found in " + maiaReposDir + ")")
	default:
		listBlock = m.renderWorktrees()
	}

	hintStyle := lipgloss.NewStyle().Foreground(ColorComment)
	hint := hintStyle.Render("↑/↓ pick worktree │ Enter create │ Esc cancel")

	content := lipgloss.JoinVertical(
		lipgloss.Left,
		title,
		"",
		listBlock,
		"",
		m.pathInput.View(),
		"",
		hint,
	)

	dialog := DialogBoxStyle.
		Width(m.dialogWidth()).
		Render(content)

	return lipgloss.Place(
		m.width,
		m.height,
		lipgloss.Center,
		lipgloss.Center,
		dialog,
	)
}

func (m *MaiaWorkerPicker) renderWorktrees() string {
	rowStyle := lipgloss.NewStyle().Foreground(ColorText).Padding(0, 1)
	selStyle := lipgloss.NewStyle().
		Foreground(ColorBg).
		Background(ColorAccent).
		Bold(true).
		Padding(0, 1)
	dimStyle := lipgloss.NewStyle().Foreground(ColorTextDim).Padding(0, 1)

	customMode := m.IsCustomPath()
	rows := make([]string, 0, len(m.worktrees))
	for i, p := range m.worktrees {
		// Show the basename (e.g. "MAIA.worker-2") — full path is in maiaReposDir.
		display := filepath.Base(p)
		switch {
		case customMode:
			// Typed path overrides selection; dim the list to signal that.
			rows = append(rows, dimStyle.Render(display))
		case i == m.cursor:
			rows = append(rows, selStyle.Render(display))
		default:
			rows = append(rows, rowStyle.Render(display))
		}
	}
	return strings.Join(rows, "\n")
}

func (m *MaiaWorkerPicker) dialogWidth() int {
	const preferred = 60
	if m.width > 0 && m.width < preferred+10 {
		w := m.width - 10
		if w < 40 {
			w = 40
		}
		return w
	}
	return preferred
}

