package views

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/roramirez/pitlist/internal/model"
)

func TestNewFilterViewDefaults(t *testing.T) {
	v := NewFilterView()
	if v.IsActive() {
		t.Error("filter should not be active initially")
	}
	if !v.showTodo {
		t.Error("showTodo should default to true")
	}
	if !v.showProgress {
		t.Error("showProgress should default to true")
	}
	if v.showDone {
		t.Error("showDone should default to false")
	}
}

func TestFilterViewActivate(t *testing.T) {
	v := NewFilterView()
	_ = v.Activate()
	if !v.IsActive() {
		t.Error("expected active after Activate()")
	}
	if v.focusIdx != 0 {
		t.Errorf("focusIdx should be 0 after Activate, got %d", v.focusIdx)
	}
}

func TestFilterViewEsc(t *testing.T) {
	v := NewFilterView()
	_ = v.Activate()

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyEsc})
	v = v2
	if v.IsActive() {
		t.Error("expected inactive after esc")
	}
}

func TestFilterViewTab(t *testing.T) {
	v := NewFilterView()
	_ = v.Activate()

	// Tab cycles focus: 0 → 1 → 2 → 0
	for _, want := range []int{1, 2, 0} {
		v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyTab})
		v = v2
		if v.focusIdx != want {
			t.Errorf("focusIdx = %d, want %d", v.focusIdx, want)
		}
	}
}

func TestFilterViewStatusToggles(t *testing.T) {
	v := NewFilterView()
	_ = v.Activate()

	// Navigate to status field (focusIdx == 2)
	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyTab})
	v, _ = v2, v2
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyTab})
	v = v2

	if v.focusIdx != 2 {
		t.Fatalf("expected focusIdx=2, got %d", v.focusIdx)
	}

	// Toggle done on
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	v = v2
	if !v.showDone {
		t.Error("showDone should be true after toggle")
	}

	// Toggle todo off
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	v = v2
	if v.showTodo {
		t.Error("showTodo should be false after toggle")
	}

	// Toggle in_progress off
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	v = v2
	if v.showProgress {
		t.Error("showProgress should be false after toggle")
	}
}

func TestFilterViewApply(t *testing.T) {
	v := NewFilterView()
	_ = v.Activate()

	// Apply with defaults → todo + in_progress statuses
	v2, cmd := v.Update(tea.KeyMsg{Type: tea.KeyEnter})
	v = v2
	if v.IsActive() {
		t.Error("expected inactive after enter")
	}
	if cmd == nil {
		t.Fatal("expected non-nil cmd from apply")
	}

	msg := cmd()
	applied, ok := msg.(FilterAppliedMsg)
	if !ok {
		t.Fatalf("expected FilterAppliedMsg, got %T", msg)
	}

	hasStatus := func(st model.TaskStatus) bool {
		for _, s := range applied.Filter.Statuses {
			if s == st {
				return true
			}
		}
		return false
	}
	if !hasStatus(model.StatusTodo) {
		t.Error("expected StatusTodo in applied filter")
	}
	if !hasStatus(model.StatusInProgress) {
		t.Error("expected StatusInProgress in applied filter")
	}
}

func TestFilterViewApplyAllOff(t *testing.T) {
	// When all statuses are toggled off, fallback to todo+in_progress
	v := NewFilterView()
	_ = v.Activate()

	// Go to status field
	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyTab})
	v = v2
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyTab})
	v = v2

	// Turn off todo and in_progress
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	v = v2
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	v = v2

	_, cmd := v.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected cmd")
	}
	msg := cmd()
	applied, ok := msg.(FilterAppliedMsg)
	if !ok {
		t.Fatalf("expected FilterAppliedMsg, got %T", msg)
	}
	if len(applied.Filter.Statuses) != 2 {
		t.Errorf("fallback should produce 2 statuses, got %d", len(applied.Filter.Statuses))
	}
}

func TestFilterViewInactiveIgnoresKeys(t *testing.T) {
	v := NewFilterView()
	// Not active — update should be a no-op
	v2, cmd := v.Update(tea.KeyMsg{Type: tea.KeyEnter})
	v = v2
	if cmd != nil {
		t.Error("inactive filter should return nil cmd")
	}
	if v.IsActive() {
		t.Error("should remain inactive")
	}
}
