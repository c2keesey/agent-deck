package ui

import (
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/asheshgoplani/agent-deck/internal/ideas"
)

// appendIdeaFunc persists a captured idea. A package var so tests can swap it
// out instead of writing to the real backlog. (local fork)
var appendIdeaFunc = ideas.AppendIdea

// IdeaDialog is a minimal one-line prompt for capturing a "by the way" idea
// straight from the dashboard — the in-view twin of the Ctrl+Alt+I popup that
// fires from inside a session. It holds only the text input plus a label for
// the session the idea is being attached to; the Home model owns persistence
// (ideas.AppendIdea) and context snapshotting, so this stays a dumb view.
// (local fork)
type IdeaDialog struct {
	visible       bool
	input         textinput.Model
	sessionLabel  string // context shown in the hint ("" when no session selected)
	width, height int
}

// NewIdeaDialog builds the dialog with its single text input.
func NewIdeaDialog() *IdeaDialog {
	ti := textinput.New()
	ti.Placeholder = "what's the idea?"
	ti.Prompt = "▸ "
	ti.Width = 50
	return &IdeaDialog{input: ti}
}

// Show opens the dialog for the given session label (may be empty), clearing any
// previous text and focusing the input.
func (d *IdeaDialog) Show(sessionLabel string) {
	d.sessionLabel = sessionLabel
	d.input.SetValue("")
	d.input.CursorEnd()
	d.input.Focus()
	d.visible = true
}

// Hide closes the dialog and blurs the input.
func (d *IdeaDialog) Hide() {
	d.visible = false
	d.input.Blur()
}

// IsVisible reports whether the dialog is open.
func (d *IdeaDialog) IsVisible() bool { return d.visible }

// Value returns the trimmed idea text.
func (d *IdeaDialog) Value() string { return strings.TrimSpace(d.input.Value()) }

// SetSize records the terminal dimensions for centering.
func (d *IdeaDialog) SetSize(width, height int) {
	d.width, d.height = width, height
}

// Update feeds key input to the text field.
func (d *IdeaDialog) Update(msg tea.KeyMsg) (*IdeaDialog, tea.Cmd) {
	var cmd tea.Cmd
	d.input, cmd = d.input.Update(msg)
	return d, cmd
}

// View renders the centered dialog box (mirrors GroupDialog styling).
func (d *IdeaDialog) View() string {
	if !d.visible {
		return ""
	}

	dialogWidth := 60
	if d.width > 0 && d.width < dialogWidth+10 {
		dialogWidth = d.width - 10
		if dialogWidth < 30 {
			dialogWidth = 30
		}
	}
	titleWidth := dialogWidth - 4

	titleStyle := DialogTitleStyle.Width(titleWidth)
	hintStyle := lipgloss.NewStyle().Foreground(ColorComment)

	context := ""
	if d.sessionLabel != "" {
		context = lipgloss.NewStyle().Foreground(ColorCyan).Render("from: "+d.sessionLabel) + "\n\n"
	}

	dialogContent := lipgloss.JoinVertical(
		lipgloss.Center,
		titleStyle.Render("Capture Idea"),
		"",
		context+d.input.View(),
		"",
		hintStyle.Render("Enter save │ Esc cancel"),
	)

	dialog := DialogBoxStyle.
		Width(dialogWidth).
		Render(dialogContent)

	return lipgloss.Place(
		d.width,
		d.height,
		lipgloss.Center,
		lipgloss.Center,
		dialog,
	)
}
