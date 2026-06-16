package ui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/asheshgoplani/agent-deck/internal/ideas"
	"github.com/asheshgoplani/agent-deck/internal/session"
)

// TestIdeaDialog_HotkeyOpensWithSelectedContext verifies Alt+I opens the
// capture dialog from the dashboard and snapshots the selected session.
func TestIdeaDialog_HotkeyOpensWithSelectedContext(t *testing.T) {
	inst := &session.Instance{ID: "s1", Title: "worker-2", ProjectPath: "/tmp/p", Tool: "claude"}
	items := []session.Item{{Type: session.ItemTypeSession, Session: inst, Level: 0}}
	h := newTestHomeWithItems(100, 30, items)
	h.cursor = 0
	h.instances = []*session.Instance{inst}
	if h.instanceByID == nil {
		h.instanceByID = map[string]*session.Instance{}
	}
	h.instanceByID[inst.ID] = inst

	model, _ := h.handleMainKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'i'}, Alt: true})
	h2 := model.(*Home)
	if !h2.ideaDialog.IsVisible() {
		t.Fatalf("Ctrl+Alt+I should open the idea dialog")
	}
	if h2.ideaCaptureSessionID != "s1" {
		t.Errorf("idea capture should snapshot the selected session, got %q", h2.ideaCaptureSessionID)
	}
}

// TestIdeaDialog_EnterSavesEntryWithContext drives the full flow and asserts the
// idea is persisted with the selected session's context. appendIdeaFunc is
// stubbed so the real backlog file is never touched.
func TestIdeaDialog_EnterSavesEntryWithContext(t *testing.T) {
	var saved *ideas.IdeaEntry
	orig := appendIdeaFunc
	appendIdeaFunc = func(e ideas.IdeaEntry) error { saved = &e; return nil }
	t.Cleanup(func() { appendIdeaFunc = orig })

	inst := &session.Instance{ID: "s1", Title: "worker-2", ProjectPath: "/tmp/p", Tool: "codex"}
	h := &Home{instanceByID: map[string]*session.Instance{"s1": inst}}
	h.instances = []*session.Instance{inst}
	h.ideaCaptureSessionID = "s1"
	h.ideaDialog = NewIdeaDialog()
	h.ideaDialog.Show("worker-2")

	// Type the idea, then Enter.
	for _, r := range "fix the flaky test" {
		h.ideaDialog, _ = h.ideaDialog.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	h.handleIdeaDialogKey(tea.KeyMsg{Type: tea.KeyEnter})

	if h.ideaDialog.IsVisible() {
		t.Errorf("dialog should close after Enter")
	}
	if saved == nil {
		t.Fatalf("Enter should have persisted an idea")
	}
	if saved.Text != "fix the flaky test" {
		t.Errorf("saved idea text = %q", saved.Text)
	}
	if saved.Project != "/tmp/p" || saved.Tool != "codex" || saved.Session != "worker-2" {
		t.Errorf("saved idea context = %+v, want project/tool/session from selected instance", *saved)
	}
}

// TestIdeaDialog_EscCancelsWithoutSaving verifies Esc discards the draft.
func TestIdeaDialog_EscCancelsWithoutSaving(t *testing.T) {
	saved := false
	orig := appendIdeaFunc
	appendIdeaFunc = func(ideas.IdeaEntry) error { saved = true; return nil }
	t.Cleanup(func() { appendIdeaFunc = orig })

	h := &Home{}
	h.ideaDialog = NewIdeaDialog()
	h.ideaDialog.Show("")
	h.ideaDialog, _ = h.ideaDialog.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	h.handleIdeaDialogKey(tea.KeyMsg{Type: tea.KeyEsc})

	if h.ideaDialog.IsVisible() {
		t.Errorf("Esc should close the dialog")
	}
	if saved {
		t.Errorf("Esc must not persist anything")
	}
}

// TestIdeaDialog_BuildEntryNoSession records a bare idea when nothing is selected.
func TestIdeaDialog_BuildEntryNoSession(t *testing.T) {
	h := &Home{instanceByID: map[string]*session.Instance{}}
	h.ideaCaptureSessionID = ""
	now := time.Date(2026, 6, 16, 12, 0, 0, 0, time.UTC)
	entry := h.buildIdeaEntry("a stray thought", now)
	if entry.Text != "a stray thought" || entry.Session != "" || entry.Project != "" {
		t.Errorf("bare idea entry = %+v, want text only", entry)
	}
}
