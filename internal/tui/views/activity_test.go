package views

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/roramirez/pitlist/internal/model"
	"github.com/roramirez/pitlist/internal/storage"
)

func newTestActivityStore(t *testing.T) *storage.YAMLStore {
	t.Helper()
	s, err := storage.NewYAMLStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewYAMLStore: %v", err)
	}
	return s
}

func TestNewActivityView(t *testing.T) {
	store := newTestActivityStore(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewActivityView(store, date)

	if v.IsInputActive() {
		t.Error("IsInputActive should be false initially")
	}
	if v.log == nil {
		t.Error("log should be initialized")
	}
}

func TestActivityViewActivityMsg(t *testing.T) {
	store := newTestActivityStore(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewActivityView(store, date)

	log := &model.ActivityLog{
		Date: date,
		Entries: []model.ActivityEntry{
			{ID: "a-20260518-001", Timestamp: date, Description: "Did work"},
		},
	}

	v2, _ := v.Update(ActivityMsg{Log: log})
	v = v2
	if len(v.log.Entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(v.log.Entries))
	}
	if v.log.Entries[0].Description != "Did work" {
		t.Errorf("unexpected description: %q", v.log.Entries[0].Description)
	}
}

func TestActivityViewWindowSize(t *testing.T) {
	store := newTestActivityStore(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewActivityView(store, date)

	v2, _ := v.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	v = v2
	if v.width != 120 || v.height != 40 {
		t.Errorf("width/height not set: %dx%d", v.width, v.height)
	}
}

func TestActivityViewNavigateJK(t *testing.T) {
	store := newTestActivityStore(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewActivityView(store, date)

	ts := time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC)
	log := &model.ActivityLog{
		Date: date,
		Entries: []model.ActivityEntry{
			{ID: "a-20260518-001", Timestamp: ts, Description: "Entry 1"},
			{ID: "a-20260518-002", Timestamp: ts, Description: "Entry 2"},
		},
	}
	v2, _ := v.Update(ActivityMsg{Log: log})
	v = v2

	// j moves down
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	v = v2
	if v.cursor != 1 {
		t.Errorf("j: expected cursor=1, got %d", v.cursor)
	}

	// j at end stays
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	v = v2
	if v.cursor != 1 {
		t.Errorf("j at end: cursor should stay at 1, got %d", v.cursor)
	}

	// k moves up
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	v = v2
	if v.cursor != 0 {
		t.Errorf("k: expected cursor=0, got %d", v.cursor)
	}
}

func TestActivityViewDayNavigation(t *testing.T) {
	store := newTestActivityStore(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewActivityView(store, date)

	// l moves to next day
	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	v = v2
	if !v.date.Equal(date.AddDate(0, 0, 1)) {
		t.Errorf("l: expected %v, got %v", date.AddDate(0, 0, 1), v.date)
	}

	// h moves to previous day
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	v = v2
	if !v.date.Equal(date) {
		t.Errorf("h: expected %v, got %v", date, v.date)
	}
}

func TestActivityViewOpenForm(t *testing.T) {
	store := newTestActivityStore(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewActivityView(store, date)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	v = v2
	if !v.IsInputActive() {
		t.Error("IsInputActive should be true after 'a'")
	}
}

func TestActivityViewFormEsc(t *testing.T) {
	store := newTestActivityStore(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewActivityView(store, date)

	// Open form
	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	v = v2

	// Close form with esc
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyEsc})
	v = v2
	if v.IsInputActive() {
		t.Error("IsInputActive should be false after esc")
	}
}

func TestActivityViewDeleteEntry(t *testing.T) {
	store := newTestActivityStore(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	ts := time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC)

	al := &model.ActivityLog{
		Date: date,
		Entries: []model.ActivityEntry{
			{ID: "a-20260518-001", Timestamp: ts, Description: "Entry 1"},
			{ID: "a-20260518-002", Timestamp: ts, Description: "Entry 2"},
		},
	}
	store.SaveActivityLog(al)

	v := NewActivityView(store, date)
	log := al
	v2, _ := v.Update(ActivityMsg{Log: log})
	v = v2

	// D deletes current entry (cursor=0)
	_, cmd := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'D'}})
	if cmd == nil {
		t.Fatal("D should return a cmd")
	}
	msg := cmd()
	am, ok := msg.(ActivityMsg)
	if !ok {
		t.Fatalf("expected ActivityMsg, got %T", msg)
	}
	if len(am.Log.Entries) != 1 {
		t.Errorf("expected 1 entry after delete, got %d", len(am.Log.Entries))
	}
}

func TestActivityViewFormTabCycling(t *testing.T) {
	store := newTestActivityStore(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewActivityView(store, date)

	// Open form
	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	v = v2

	// Tab through fields
	for i := 1; i <= actFormFields; i++ {
		v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyTab})
		v = v2
		want := i % actFormFields
		if v.form.focusIdx != want {
			t.Errorf("after %d tabs: focusIdx=%d, want %d", i, v.form.focusIdx, want)
		}
	}
}

func TestActivityViewFormShiftTab(t *testing.T) {
	store := newTestActivityStore(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewActivityView(store, date)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	v = v2

	// Shift+Tab goes back (wraps from 0 to last)
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	v = v2
	if v.form.focusIdx != actFormFields-1 {
		t.Errorf("shift+tab: focusIdx=%d, want %d", v.form.focusIdx, actFormFields-1)
	}
}

func TestActivityViewFormCtrlS(t *testing.T) {
	store := newTestActivityStore(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewActivityView(store, date)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	v = v2

	// Type a description into the description field
	for _, ch := range []rune{'W', 'o', 'r', 'k'} {
		v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		v = v2
	}

	// ctrl+s submits the form
	v2, cmd := v.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	v = v2
	if !v.form.active {
		// form was closed – good
	}
	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(ActivityMsg); !ok {
			t.Errorf("expected ActivityMsg after submit, got %T", msg)
		}
	}
}

func TestActivityViewFormEnterLastField(t *testing.T) {
	store := newTestActivityStore(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewActivityView(store, date)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	v = v2

	// Type description
	for _, ch := range []rune{'D', 'o', 'n', 'e'} {
		v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		v = v2
	}

	// Tab to last field (taskRef = index 4)
	for i := 0; i < actFormFields-1; i++ {
		v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyTab})
		v = v2
	}
	if v.form.focusIdx != actFormFields-1 {
		t.Fatalf("expected focus on last field (%d), got %d", actFormFields-1, v.form.focusIdx)
	}

	// enter on last field → submit
	v2, cmd := v.Update(tea.KeyMsg{Type: tea.KeyEnter})
	v = v2
	if cmd != nil {
		msg := cmd()
		if _, ok := msg.(ActivityMsg); !ok {
			t.Errorf("expected ActivityMsg after enter-submit, got %T", msg)
		}
	}
}

func TestActivityViewFormDefaultKey(t *testing.T) {
	store := newTestActivityStore(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewActivityView(store, date)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	v = v2

	// Type normally into each field via tab+key
	for _, field := range []int{0, 1, 2, 3, 4} {
		_ = field
		v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
		v = v2
		v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyTab})
		v = v2
	}
}

func TestActivityViewView(t *testing.T) {
	store := newTestActivityStore(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	ts := time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC)
	v := NewActivityView(store, date)

	al := &model.ActivityLog{
		Date: date,
		Entries: []model.ActivityEntry{
			{ID: "a-001", Timestamp: ts, Description: "Entry", Tags: []string{"work"}, DurationMin: 30, TaskRef: "t-001"},
		},
	}
	v2, _ := v.Update(ActivityMsg{Log: al})
	v = v2

	out := v.View(120, 40)
	if out == "" {
		t.Error("View returned empty string")
	}
}

func TestActivityViewFormView(t *testing.T) {
	store := newTestActivityStore(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewActivityView(store, date)

	// Open form and render
	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	v = v2

	out := v.View(120, 40)
	if out == "" {
		t.Error("View with form open returned empty")
	}
}

func TestActivityViewLoad(t *testing.T) {
	store := newTestActivityStore(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)

	// Save an entry to store
	ts := time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC)
	al := &model.ActivityLog{
		Date: date,
		Entries: []model.ActivityEntry{
			{ID: "a-20260518-001", Timestamp: ts, Description: "Loaded entry"},
		},
	}
	if err := store.SaveActivityLog(al); err != nil {
		t.Fatal(err)
	}

	v := NewActivityView(store, date)
	cmd := v.Load()
	if cmd == nil {
		t.Fatal("Load should return a cmd")
	}

	msg := cmd()
	actMsg, ok := msg.(ActivityMsg)
	if !ok {
		t.Fatalf("expected ActivityMsg, got %T", msg)
	}
	if len(actMsg.Log.Entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(actMsg.Log.Entries))
	}
}

// ── submitForm with taskRef ───────────────────────────────────────────────────

func TestActivityViewSubmitFormWithRef(t *testing.T) {
	store := newTestActivityStore(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	taskDate := date
	plan := &model.DayPlan{
		Date: taskDate,
		Tasks: []model.Task{
			{ID: "t-20260518-001", Title: "Ref task", Status: model.StatusTodo,
				CreatedAt: taskDate, UpdatedAt: taskDate},
		},
	}
	if err := store.SaveDayPlan(plan); err != nil {
		t.Fatal(err)
	}

	v := NewActivityView(store, date)

	// Open form
	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	v = v2

	// Type description "Work"
	for _, ch := range []rune{'W', 'o', 'r', 'k'} {
		v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		v = v2
	}

	// Tab to taskRef field (index 4 = actFormFields-1)
	for i := 0; i < actFormFields-1; i++ {
		v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyTab})
		v = v2
	}
	if v.form.focusIdx != actFormFields-1 {
		t.Fatalf("expected focusIdx=%d (taskRef), got %d", actFormFields-1, v.form.focusIdx)
	}

	// Type a task ID
	for _, ch := range []rune{'t', '-', '2', '0', '2', '6', '0', '5', '1', '8', '-', '0', '0', '1'} {
		v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		v = v2
	}

	// ctrl+s submit
	v2, cmd := v.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	v = v2
	if cmd == nil {
		t.Fatal("ctrl+s with description should return a cmd")
	}
	msg := cmd()
	am, ok := msg.(ActivityMsg)
	if !ok {
		t.Fatalf("expected ActivityMsg after submit, got %T", msg)
	}
	if len(am.Log.Entries) == 0 {
		t.Fatal("expected at least 1 entry after submit")
	}
	if am.Log.Entries[len(am.Log.Entries)-1].TaskRef != "t-20260518-001" {
		t.Errorf("TaskRef = %q, want t-20260518-001", am.Log.Entries[len(am.Log.Entries)-1].TaskRef)
	}
}

// ── deleteEntry idx out of bounds ────────────────────────────────────────────

func TestActivityViewDeleteEntryOutOfBounds(t *testing.T) {
	store := newTestActivityStore(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	ts := time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC)

	al := &model.ActivityLog{
		Date: date,
		Entries: []model.ActivityEntry{
			{ID: "a-001", Timestamp: ts, Description: "Only"},
		},
	}
	store.SaveActivityLog(al)

	v := NewActivityView(store, date)
	v2, _ := v.Update(ActivityMsg{Log: al})
	v = v2

	// deleteEntry with idx=-1 (out of bounds) returns ActivityMsg unchanged
	cmd := v.deleteEntry(-1)
	if cmd == nil {
		t.Fatal("expected cmd even for out-of-bounds")
	}
	msg := cmd()
	am, ok := msg.(ActivityMsg)
	if !ok {
		t.Fatalf("expected ActivityMsg, got %T", msg)
	}
	if len(am.Log.Entries) != 1 {
		t.Errorf("out-of-bounds delete should not remove entry, got %d entries", len(am.Log.Entries))
	}
}

// ── submitForm with taskRef ───────────────────────────────────────────────────

func TestActivityViewSubmitFormWithTaskRef(t *testing.T) {
	store := newTestActivityStore(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	// Seed a task so AddActivityRefToTask works
	taskPlan := &model.DayPlan{
		Date: date,
		Tasks: []model.Task{
			{ID: "t-20260518-001", Title: "Ref task", Status: model.StatusTodo, CreatedAt: date, UpdatedAt: date},
		},
	}
	store.SaveDayPlan(taskPlan)

	v := NewActivityView(store, date)
	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	v = v2

	// Type description
	for _, ch := range []rune{'W', 'o', 'r', 'k'} {
		v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		v = v2
	}
	// Tab to taskRef field (index 4)
	for i := 0; i < actFormFields-1; i++ {
		v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyTab})
		v = v2
	}
	// Type task ID into taskRef field
	for _, ch := range []rune{'t', '-', '2', '0', '2', '6', '0', '5', '1', '8', '-', '0', '0', '1'} {
		v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		v = v2
	}

	// ctrl+s to submit
	v2, cmd := v.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	v = v2
	if cmd == nil {
		t.Fatal("expected cmd after submit with ref")
	}
	msg := cmd()
	if _, ok := msg.(ActivityMsg); !ok {
		t.Fatalf("expected ActivityMsg, got %T", msg)
	}
}

// ── updateForm enter on non-last field ───────────────────────────────────────

func TestActivityViewFormEnterNonLastField(t *testing.T) {
	store := newTestActivityStore(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewActivityView(store, date)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	v = v2

	// enter on field 0 → advance to field 1
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyEnter})
	v = v2
	if v.form.focusIdx != 1 {
		t.Errorf("enter on field 0: focusIdx=%d, want 1", v.form.focusIdx)
	}
}
