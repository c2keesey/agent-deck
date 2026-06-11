package ui

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
)

func TestMaiaWorkerPicker_ShellHint(t *testing.T) {
	p := &MaiaWorkerPicker{
		visible: true,
		workers: []string{"/r/MAIA.worker-1"},
		width:   80,
		height:  24,
	}
	view := p.View()
	if !strings.Contains(view, "s shell") {
		t.Errorf("picker hint should advertise the raw-shell option; got:\n%s", view)
	}
	if !strings.Contains(view, "c codex") {
		t.Errorf("picker hint should advertise the codex option; got:\n%s", view)
	}
	if !strings.Contains(view, "~ home") {
		t.Errorf("picker hint should advertise the home-root option; got:\n%s", view)
	}
}

func TestMaiaWorkerPicker_NextOpenWorker(t *testing.T) {
	w := []string{"/r/MAIA.worker-1", "/r/MAIA.worker-2", "/r/MAIA.worker-3"}

	// worker-1 occupied -> next open is index 1 (worker-2).
	p := &MaiaWorkerPicker{workers: w, occupied: map[string]bool{"/r/MAIA.worker-1": true}}
	if got := p.nextOpenWorker(); got != 1 {
		t.Errorf("nextOpenWorker = %d, want 1", got)
	}

	// none occupied -> index 0.
	p = &MaiaWorkerPicker{workers: w, occupied: map[string]bool{}}
	if got := p.nextOpenWorker(); got != 0 {
		t.Errorf("nextOpenWorker (none occupied) = %d, want 0", got)
	}

	// all occupied -> fall back to 0.
	p = &MaiaWorkerPicker{workers: w, occupied: map[string]bool{
		"/r/MAIA.worker-1": true, "/r/MAIA.worker-2": true, "/r/MAIA.worker-3": true,
	}}
	if got := p.nextOpenWorker(); got != 0 {
		t.Errorf("nextOpenWorker (all occupied) = %d, want 0", got)
	}
}

func TestMaiaWorkerPicker_SelectedPerColumn(t *testing.T) {
	p := &MaiaWorkerPicker{
		workers:      []string{"/r/MAIA.worker-1", "/r/MAIA.worker-2"},
		roDevs:       []string{"/r/MAIA.ro-dev"},
		workerCursor: 1,
	}

	if path, group := p.Selected(); path != "/r/MAIA.worker-2" || group != maiaWorkerGroup {
		t.Errorf("worker Selected = (%q, %q), want (worker-2, %q)", path, group, maiaWorkerGroup)
	}

	p.focusCol = 1
	if path, group := p.Selected(); path != "/r/MAIA.ro-dev" || group != maiaRoDevGroup {
		t.Errorf("ro-dev Selected = (%q, %q), want (ro-dev, %q)", path, group, maiaRoDevGroup)
	}
}

func TestMaiaWorkerPicker_Navigation(t *testing.T) {
	p := &MaiaWorkerPicker{
		workers: []string{"/r/MAIA.worker-1", "/r/MAIA.worker-2"},
		roDevs:  []string{"/r/MAIA.ro-dev"},
	}
	key := func(t tea.KeyType) tea.KeyMsg { return tea.KeyMsg{Type: t} }
	runeKey := func(r rune) tea.KeyMsg { return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}} }

	// 'r' switches to the ro-dev column.
	p, _ = p.Update(runeKey('r'))
	if p.focusCol != 1 {
		t.Fatalf("after r, focusCol = %d, want 1", p.focusCol)
	}
	// Down is clamped (only one ro-dev).
	p, _ = p.Update(key(tea.KeyDown))
	if p.roDevCursor != 0 {
		t.Fatalf("roDevCursor = %d, want 0 (clamped)", p.roDevCursor)
	}
	// 'r' toggles back to workers; down advances within range.
	p, _ = p.Update(runeKey('r'))
	p, _ = p.Update(key(tea.KeyDown))
	if p.focusCol != 0 || p.workerCursor != 1 {
		t.Fatalf("after r+down: focusCol=%d workerCursor=%d, want 0,1", p.focusCol, p.workerCursor)
	}
}

func TestWorkerSortKey(t *testing.T) {
	if a, b := workerSortKey("/r/MAIA.worker-2"), workerSortKey("/r/MAIA.worker-10"); a >= b {
		t.Errorf("worker-2 (%d) should sort before worker-10 (%d)", a, b)
	}
	if a, b := workerSortKey("/r/MAIA.worker-9"), workerSortKey("/r/MAIA.worker-retry"); a >= b {
		t.Errorf("worker-9 (%d) should sort before worker-retry (%d)", a, b)
	}
}
