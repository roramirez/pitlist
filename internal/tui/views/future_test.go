package views

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/roramirez/pitlist/internal/model"
	"github.com/roramirez/pitlist/internal/storage"
)

func newFutureStore(t *testing.T) *storage.YAMLStore {
	t.Helper()
	s, err := storage.NewYAMLStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewYAMLStore: %v", err)
	}
	return s
}

func seedFutureTask(t *testing.T, s *storage.YAMLStore, tasks ...model.Task) {
	t.Helper()
	list := &model.FutureList{Tasks: tasks}
	if err := s.SaveFutureList(list); err != nil {
		t.Fatalf("SaveFutureList: %v", err)
	}
}

func futureViewWithTask(t *testing.T) (FutureView, *storage.YAMLStore) {
	t.Helper()
	s := newFutureStore(t)
	now := time.Now().UTC()
	task := model.Task{ID: "f-20260525-001", Title: "My future task", Status: model.StatusTodo, CreatedAt: now, UpdatedAt: now}
	seedFutureTask(t, s, task)

	v := NewFutureView(s, "work", "personal")
	list, _ := s.GetFutureList()
	v2, _ := v.Update(FutureMsg{List: list})
	return v2, s
}

// ── constructor & load ────────────────────────────────────────────────────────

func TestNewFutureView(t *testing.T) {
	s := newFutureStore(t)
	v := NewFutureView(s, "work")
	if v.IsInputActive() {
		t.Error("IsInputActive should be false initially")
	}
	if v.list == nil {
		t.Error("list should be initialized")
	}
}

func TestFutureViewLoad(t *testing.T) {
	s := newFutureStore(t)
	v := NewFutureView(s)
	cmd := v.Load()
	if cmd == nil {
		t.Fatal("Load should return a cmd")
	}
	msg := cmd()
	if _, ok := msg.(FutureMsg); !ok {
		t.Fatalf("expected FutureMsg, got %T", msg)
	}
}

func TestFutureViewUpdateFutureMsg(t *testing.T) {
	s := newFutureStore(t)
	now := time.Now().UTC()
	v := NewFutureView(s)

	list := &model.FutureList{
		Tasks: []model.Task{
			{ID: "f-001", Title: "A", Status: model.StatusTodo, CreatedAt: now, UpdatedAt: now},
			{ID: "f-002", Title: "B", Status: model.StatusTodo, CreatedAt: now, UpdatedAt: now},
		},
	}
	v2, _ := v.Update(FutureMsg{List: list})
	v = v2
	if len(v.list.Tasks) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(v.list.Tasks))
	}
}

func TestFutureViewUpdateWindowSize(t *testing.T) {
	s := newFutureStore(t)
	v := NewFutureView(s)

	v2, _ := v.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	v = v2
	if v.width != 120 || v.height != 40 {
		t.Errorf("size not set: %dx%d", v.width, v.height)
	}
}

func TestFutureViewFutureMsgCursorClamp(t *testing.T) {
	s := newFutureStore(t)
	now := time.Now().UTC()
	v := NewFutureView(s)
	v.cursor = 5

	list := &model.FutureList{Tasks: []model.Task{
		{ID: "f-001", Title: "Only one", Status: model.StatusTodo, CreatedAt: now, UpdatedAt: now},
	}}
	v2, _ := v.Update(FutureMsg{List: list})
	v = v2
	if v.cursor != 0 {
		t.Errorf("cursor should clamp to 0, got %d", v.cursor)
	}
}

func TestFutureViewLinkedActivitiesMsg(t *testing.T) {
	s := newFutureStore(t)
	v := NewFutureView(s)

	ts := time.Now().UTC()
	entries := []model.ActivityEntry{
		{ID: "a-001", Timestamp: ts, Description: "Work on it"},
	}
	v2, _ := v.Update(FutureLinkedActivitiesMsg{Entries: entries})
	v = v2
	if len(v.linkedActivities) != 1 {
		t.Errorf("expected 1 linked activity, got %d", len(v.linkedActivities))
	}
}

// ── navigation ────────────────────────────────────────────────────────────────

func TestFutureViewNavigateJK(t *testing.T) {
	v, _ := futureViewWithTask(t)
	s := newFutureStore(t)
	now := time.Now().UTC()
	seedFutureTask(t, s, model.Task{ID: "f-001", Title: "A", Status: model.StatusTodo, CreatedAt: now, UpdatedAt: now},
		model.Task{ID: "f-002", Title: "B", Status: model.StatusTodo, CreatedAt: now, UpdatedAt: now})
	v2 := NewFutureView(s)
	list, _ := s.GetFutureList()
	v2u, _ := v2.Update(FutureMsg{List: list})
	v = v2u

	v3, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	v = v3
	if v.cursor != 1 {
		t.Errorf("j: cursor=%d, want 1", v.cursor)
	}

	v3, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	v = v3
	if v.cursor != 0 {
		t.Errorf("k: cursor=%d, want 0", v.cursor)
	}
}

func TestFutureViewNavigateBounds(t *testing.T) {
	s := newFutureStore(t)
	now := time.Now().UTC()
	seedFutureTask(t, s, model.Task{ID: "f-001", Title: "Only", Status: model.StatusTodo, CreatedAt: now, UpdatedAt: now})
	v := NewFutureView(s)
	list, _ := s.GetFutureList()
	vu, _ := v.Update(FutureMsg{List: list})
	v = vu

	// k at top → no change
	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	v = v2
	if v.cursor != 0 {
		t.Errorf("k at top: cursor=%d, want 0", v.cursor)
	}

	// j at bottom → no change
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	v = v2
	if v.cursor != 0 {
		t.Errorf("j at bottom: cursor=%d, want 0", v.cursor)
	}
}

func TestFutureViewPaneTab(t *testing.T) {
	s := newFutureStore(t)
	v := NewFutureView(s)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyTab})
	v = v2
	if v.pane != 1 {
		t.Errorf("tab: pane=%d, want 1", v.pane)
	}
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyTab})
	v = v2
	if v.pane != 0 {
		t.Errorf("second tab: pane=%d, want 0", v.pane)
	}
}

// ── add form ──────────────────────────────────────────────────────────────────

func TestFutureViewOpenAddForm(t *testing.T) {
	s := newFutureStore(t)
	v := NewFutureView(s)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	v = v2
	if !v.adding {
		t.Error("'a' should open add form")
	}
	if !v.IsInputActive() {
		t.Error("IsInputActive should be true when adding")
	}

	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyEsc})
	v = v2
	if v.adding {
		t.Error("esc should close add form")
	}
}

func TestFutureViewAddFormSave(t *testing.T) {
	s := newFutureStore(t)
	v := NewFutureView(s, "work")

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	v = v2

	for _, ch := range []rune{'S', 'o', 'm', 'e', 'd', 'a', 'y'} {
		v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		v = v2
	}

	v2, cmd := v.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	v = v2
	if cmd == nil {
		t.Fatal("ctrl+s with title should return a cmd")
	}
	if v.adding {
		t.Error("adding should be false after save")
	}
	msg := cmd()
	if _, ok := msg.(FutureMsg); !ok {
		t.Fatalf("expected FutureMsg after add, got %T", msg)
	}

	list, _ := s.GetFutureList()
	if len(list.Tasks) != 1 || list.Tasks[0].Title != "Someday" {
		t.Errorf("task not saved: %v", list.Tasks)
	}
}

func TestFutureViewAddFormEmptyCtrlS(t *testing.T) {
	s := newFutureStore(t)
	v := NewFutureView(s)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	v = v2

	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	v = v2
	if v.adding {
		t.Error("empty title ctrl+s should close form")
	}
}

func TestFutureViewAddFormTab(t *testing.T) {
	s := newFutureStore(t)
	v := NewFutureView(s, "work")

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	v = v2

	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyTab})
	v = v2
	if v.tForm.focusIdx != 1 {
		t.Errorf("tab: focusIdx=%d, want 1", v.tForm.focusIdx)
	}

	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	v = v2
	if v.tForm.focusIdx != 0 {
		t.Errorf("shift+tab: focusIdx=%d, want 0", v.tForm.focusIdx)
	}
}

func TestFutureViewAddFormEnterLastField(t *testing.T) {
	s := newFutureStore(t)
	v := NewFutureView(s)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	v = v2

	for _, ch := range []rune{'T', 'a', 's', 'k'} {
		v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		v = v2
	}

	for i := 0; i < taskFormFields-1; i++ {
		v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyTab})
		v = v2
	}

	v2, cmd := v.Update(tea.KeyMsg{Type: tea.KeyEnter})
	v = v2
	_ = cmd
	if v.adding {
		t.Error("enter on last field should submit and close form")
	}
}

// ── toggle done ───────────────────────────────────────────────────────────────

func TestFutureViewToggleDone(t *testing.T) {
	v, s := futureViewWithTask(t)

	_, cmd := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if cmd == nil {
		t.Fatal("'d' should return a cmd")
	}
	msg := cmd()
	if _, ok := msg.(FutureMsg); !ok {
		t.Errorf("expected FutureMsg after toggle, got %T", msg)
	}

	list, _ := s.GetFutureList()
	if list.Tasks[0].Status != model.StatusDone {
		t.Errorf("expected StatusDone, got %q", list.Tasks[0].Status)
	}
}

func TestFutureViewToggleDoneToTodo(t *testing.T) {
	s := newFutureStore(t)
	now := time.Now().UTC()
	doneAt := now
	seedFutureTask(t, s, model.Task{
		ID: "f-001", Title: "Done", Status: model.StatusDone,
		DoneAt: &doneAt, CreatedAt: now, UpdatedAt: now,
	})
	v := NewFutureView(s)
	list, _ := s.GetFutureList()
	vu, _ := v.Update(FutureMsg{List: list})
	v = vu

	_, cmd := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	msg := cmd()
	if _, ok := msg.(FutureMsg); !ok {
		t.Fatalf("expected FutureMsg, got %T", msg)
	}
	updated, _ := s.GetFutureList()
	if updated.Tasks[0].Status != model.StatusTodo {
		t.Errorf("expected StatusTodo after toggle, got %q", updated.Tasks[0].Status)
	}
}

// ── delete ────────────────────────────────────────────────────────────────────

func TestFutureViewDeleteTask(t *testing.T) {
	v, s := futureViewWithTask(t)

	_, cmd := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'D'}})
	if cmd == nil {
		t.Fatal("D should return a cmd")
	}
	msg := cmd()
	if _, ok := msg.(FutureMsg); !ok {
		t.Fatalf("expected FutureMsg after delete, got %T", msg)
	}

	list, _ := s.GetFutureList()
	if len(list.Tasks) != 0 {
		t.Errorf("task should be deleted, got %d tasks", len(list.Tasks))
	}
}

// ── notes ─────────────────────────────────────────────────────────────────────

func TestFutureViewNotesEsc(t *testing.T) {
	v, _ := futureViewWithTask(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	v = v2
	if v.detailMode != futureDetailEditNotes {
		t.Fatalf("expected futureDetailEditNotes, got %v", v.detailMode)
	}
	if !v.IsInputActive() {
		t.Error("IsInputActive should be true in notes mode")
	}

	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyEsc})
	v = v2
	if v.detailMode != futureDetailNormal {
		t.Errorf("expected futureDetailNormal after esc, got %v", v.detailMode)
	}
}

func TestFutureViewNotesSave(t *testing.T) {
	v, _ := futureViewWithTask(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	v = v2

	_, cmd := v.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	if cmd == nil {
		t.Fatal("ctrl+s in notes mode should return a cmd")
	}
	msg := cmd()
	if _, ok := msg.(FutureMsg); !ok {
		t.Fatalf("expected FutureMsg after notes save, got %T", msg)
	}
}

func TestFutureViewNotesDefaultKey(t *testing.T) {
	v, _ := futureViewWithTask(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	v = v2

	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	v = v2
	if v.detailMode != futureDetailEditNotes {
		t.Errorf("should stay in notes mode, got %v", v.detailMode)
	}
}

// ── edit task form ────────────────────────────────────────────────────────────

func TestFutureViewEditFormEsc(t *testing.T) {
	v, _ := futureViewWithTask(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	v = v2
	if v.detailMode != futureDetailEditTask {
		t.Fatalf("expected futureDetailEditTask, got %v", v.detailMode)
	}

	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyEsc})
	v = v2
	if v.detailMode != futureDetailNormal {
		t.Errorf("expected futureDetailNormal after esc")
	}
}

func TestFutureViewEditFormSave(t *testing.T) {
	v, s := futureViewWithTask(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	v = v2

	v2, cmd := v.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	v = v2
	if cmd == nil {
		t.Fatal("ctrl+s in edit form should return a cmd")
	}
	msg := cmd()
	if _, ok := msg.(FutureMsg); !ok {
		t.Fatalf("expected FutureMsg after edit save, got %T", msg)
	}

	list, _ := s.GetFutureList()
	if len(list.Tasks) != 1 {
		t.Errorf("expected 1 task after edit, got %d", len(list.Tasks))
	}
}

func TestFutureViewEditFormTab(t *testing.T) {
	v, _ := futureViewWithTask(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	v = v2

	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyTab})
	v = v2
	if v.tForm.focusIdx != 1 {
		t.Errorf("tab: focusIdx=%d, want 1", v.tForm.focusIdx)
	}
}

func TestFutureViewEditFormShiftTab(t *testing.T) {
	v, _ := futureViewWithTask(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	v = v2

	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	v = v2
	if v.tForm.focusIdx != taskFormFields-1 {
		t.Errorf("shift+tab from 0: focusIdx=%d, want %d", v.tForm.focusIdx, taskFormFields-1)
	}
}

func TestFutureViewEditFormEnterLastField(t *testing.T) {
	v, _ := futureViewWithTask(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	v = v2

	for i := 0; i < taskFormFields-1; i++ {
		v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyTab})
		v = v2
	}

	v2, cmd := v.Update(tea.KeyMsg{Type: tea.KeyEnter})
	v = v2
	if cmd == nil {
		t.Fatal("enter on last edit field should return a cmd")
	}
	if _, ok := cmd().(FutureMsg); !ok {
		t.Error("expected FutureMsg after edit enter on last field")
	}
}

// ── log activity form ─────────────────────────────────────────────────────────

func TestFutureViewLogFormEsc(t *testing.T) {
	v, _ := futureViewWithTask(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'L'}})
	v = v2
	if v.detailMode != futureDetailLogActivity {
		t.Fatalf("expected futureDetailLogActivity, got %v", v.detailMode)
	}

	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyEsc})
	v = v2
	if v.detailMode != futureDetailNormal {
		t.Errorf("esc should close log form, got %v", v.detailMode)
	}
}

func TestFutureViewLogFormTab(t *testing.T) {
	v, _ := futureViewWithTask(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'L'}})
	v = v2

	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyTab})
	v = v2
	if v.logForm.focusIdx != 1 {
		t.Errorf("tab: focusIdx=%d, want 1", v.logForm.focusIdx)
	}

	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	v = v2
	if v.logForm.focusIdx != 0 {
		t.Errorf("shift+tab: focusIdx=%d, want 0", v.logForm.focusIdx)
	}
}

func TestFutureViewLogFormSubmit(t *testing.T) {
	v, _ := futureViewWithTask(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'L'}})
	v = v2

	for _, ch := range []rune{'W', 'o', 'r', 'k'} {
		v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		v = v2
	}

	v2, cmd := v.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	v = v2
	if cmd == nil {
		t.Fatal("ctrl+s with description should return a cmd")
	}
	msg := cmd()
	if _, ok := msg.(FutureMsg); !ok {
		t.Fatalf("expected FutureMsg after log submit, got %T", msg)
	}
}

func TestFutureViewLogFormEnterNonLastField(t *testing.T) {
	v, _ := futureViewWithTask(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'L'}})
	v = v2

	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyEnter})
	v = v2
	if v.logForm.focusIdx != 1 {
		t.Errorf("enter on field 0: expected focusIdx=1, got %d", v.logForm.focusIdx)
	}
}

func TestFutureViewLogFormEnterLastField(t *testing.T) {
	v, _ := futureViewWithTask(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'L'}})
	v = v2

	for _, ch := range []rune{'W', 'o', 'r', 'k'} {
		v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		v = v2
	}

	for i := 0; i < quickLogFields-1; i++ {
		v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyTab})
		v = v2
	}

	v2, cmd := v.Update(tea.KeyMsg{Type: tea.KeyEnter})
	v = v2
	_ = v
	if cmd == nil {
		t.Fatal("enter on last log field should return a cmd")
	}
}

// ── schedule ──────────────────────────────────────────────────────────────────

func TestFutureViewScheduleEsc(t *testing.T) {
	v, _ := futureViewWithTask(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	v = v2
	if v.detailMode != futureDetailSchedule {
		t.Fatalf("expected futureDetailSchedule, got %v", v.detailMode)
	}
	if !v.IsInputActive() {
		t.Error("IsInputActive should be true in schedule mode")
	}

	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyEsc})
	v = v2
	if v.detailMode != futureDetailNormal {
		t.Errorf("esc should close schedule prompt, got %v", v.detailMode)
	}
}

func TestFutureViewScheduleToday(t *testing.T) {
	v, s := futureViewWithTask(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	v = v2

	v.scheduleInput.SetValue("today")
	v2, cmd := v.Update(tea.KeyMsg{Type: tea.KeyEnter})
	v = v2
	if cmd == nil {
		t.Fatal("enter with date should return a cmd")
	}
	msg := cmd()
	if _, ok := msg.(FutureMsg); !ok {
		t.Fatalf("expected FutureMsg after schedule, got %T", msg)
	}

	// Task should be gone from future list
	list, _ := s.GetFutureList()
	if len(list.Tasks) != 0 {
		t.Errorf("task should be removed from future list, got %d tasks", len(list.Tasks))
	}
}

func TestFutureViewScheduleTomorrow(t *testing.T) {
	v, s := futureViewWithTask(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	v = v2

	v.scheduleInput.SetValue("tomorrow")
	v2, cmd := v.Update(tea.KeyMsg{Type: tea.KeyEnter})
	v = v2
	msg := cmd()
	if _, ok := msg.(FutureMsg); !ok {
		t.Fatalf("expected FutureMsg, got %T", msg)
	}

	tomorrow := time.Now().AddDate(0, 0, 1)
	day := time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 0, 0, 0, 0, time.UTC)
	plan, _ := s.GetDayPlan(day)
	if len(plan.Tasks) != 1 {
		t.Errorf("expected task on tomorrow's plan, got %d tasks", len(plan.Tasks))
	}
}

func TestFutureViewScheduleExplicitDate(t *testing.T) {
	v, s := futureViewWithTask(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	v = v2

	targetDate := time.Date(2027, 1, 15, 0, 0, 0, 0, time.UTC)
	v.scheduleInput.SetValue("2027-01-15")
	_, cmd := v.Update(tea.KeyMsg{Type: tea.KeyEnter})
	msg := cmd()
	if _, ok := msg.(FutureMsg); !ok {
		t.Fatalf("expected FutureMsg, got %T", msg)
	}

	plan, _ := s.GetDayPlan(targetDate)
	if len(plan.Tasks) != 1 {
		t.Errorf("expected task on 2027-01-15, got %d tasks", len(plan.Tasks))
	}
}

func TestFutureViewScheduleCtrlS(t *testing.T) {
	v, _ := futureViewWithTask(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	v = v2

	v.scheduleInput.SetValue("today")
	v2, cmd := v.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	v = v2
	if cmd == nil {
		t.Fatal("ctrl+s should also submit the schedule prompt")
	}
}

// ── view rendering ────────────────────────────────────────────────────────────

func TestFutureViewEmpty(t *testing.T) {
	s := newFutureStore(t)
	v := NewFutureView(s)
	out := v.View(120, 40)
	if out == "" {
		t.Error("View should return non-empty string even when empty")
	}
}

func TestFutureViewWithTask(t *testing.T) {
	v, _ := futureViewWithTask(t)
	out := v.View(120, 40)
	if out == "" {
		t.Error("View with task returned empty")
	}
}

func TestFutureViewWithAddForm(t *testing.T) {
	s := newFutureStore(t)
	v := NewFutureView(s)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	v = v2

	out := v.View(120, 40)
	if out == "" {
		t.Error("View with add form returned empty")
	}
}

func TestFutureViewWithNotesMode(t *testing.T) {
	v, _ := futureViewWithTask(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	v = v2

	out := v.View(120, 40)
	if out == "" {
		t.Error("View in notes mode returned empty")
	}
}

func TestFutureViewWithEditMode(t *testing.T) {
	v, _ := futureViewWithTask(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	v = v2

	out := v.View(120, 40)
	if out == "" {
		t.Error("View in edit mode returned empty")
	}
}

func TestFutureViewWithLogMode(t *testing.T) {
	v, _ := futureViewWithTask(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'L'}})
	v = v2

	out := v.View(120, 40)
	if out == "" {
		t.Error("View in log mode returned empty")
	}
}

func TestFutureViewWithScheduleMode(t *testing.T) {
	v, _ := futureViewWithTask(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	v = v2

	out := v.View(120, 40)
	if out == "" {
		t.Error("View in schedule mode returned empty")
	}
}

func TestFutureViewDetailPaneNoTask(t *testing.T) {
	s := newFutureStore(t)
	v := NewFutureView(s)
	v.pane = 1
	out := v.View(120, 40)
	if out == "" {
		t.Error("View with empty list in detail pane returned empty")
	}
}

func TestFutureViewWithLinkedActivities(t *testing.T) {
	v, _ := futureViewWithTask(t)

	ts := time.Now().UTC()
	v2, _ := v.Update(FutureLinkedActivitiesMsg{Entries: []model.ActivityEntry{
		{ID: "a-001", Timestamp: ts, Description: "Deep work", DurationMin: 60, Tags: []string{"focus"}},
	}})
	v = v2

	out := v.View(120, 40)
	if out == "" {
		t.Error("View with linked activities returned empty")
	}
}

func TestFutureViewSaveFutureListMsg(t *testing.T) {
	store := newFutureStore(t)
	v := NewFutureView(store)
	list := &model.FutureList{Tasks: []model.Task{
		{ID: "f-001", Title: "Future", Status: model.StatusTodo},
	}}
	msg := v.saveFutureListMsg(list)
	if _, ok := msg.(errMsg); ok {
		t.Error("saveFutureListMsg returned errMsg on success")
	}
}

func TestFutureViewHandleExistingFutureAction(t *testing.T) {
	store := newFutureStore(t)
	v := NewFutureView(store)
	t0 := model.Task{ID: "f-001", Title: "Task", Status: model.StatusTodo}

	// unknown key → no-op
	_, cmd := v.handleExistingFutureAction(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}, t0)
	if cmd != nil {
		t.Error("unknown key should return nil cmd")
	}

	// "d" → toggle done (returns non-nil cmd)
	_, cmd = v.handleExistingFutureAction(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}, t0)
	if cmd == nil {
		t.Error("'d' should return a cmd")
	}
}

// ── saveFutureListMsg error branch ────────────────────────────────────────────

func TestSaveFutureListMsgError(t *testing.T) {
	dir := t.TempDir()
	store, _ := storage.NewYAMLStore(dir)
	v := NewFutureView(store)

	if err := os.Chmod(dir, 0o555); err != nil {
		t.Fatalf("Chmod: %v", err)
	}
	t.Cleanup(func() { os.Chmod(dir, 0o755) })

	list := &model.FutureList{}
	msg := v.saveFutureListMsg(list)
	if _, ok := msg.(errMsg); !ok {
		t.Fatalf("expected errMsg on write failure, got %T", msg)
	}
}

// ── loadLinkedActivities ──────────────────────────────────────────────────────

func TestLoadLinkedActivitiesEmptyID(t *testing.T) {
	s := newFutureStore(t)
	v := NewFutureView(s)

	cmd := v.loadLinkedActivities("")
	if cmd != nil {
		t.Error("expected nil cmd for empty taskID")
	}
}

func TestLoadLinkedActivitiesTaskNotFound(t *testing.T) {
	s := newFutureStore(t)
	v := NewFutureView(s)

	cmd := v.loadLinkedActivities("no-such-id")
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	msg := cmd()
	m, ok := msg.(FutureLinkedActivitiesMsg)
	if !ok {
		t.Fatalf("expected FutureLinkedActivitiesMsg, got %T", msg)
	}
	if len(m.Entries) != 0 {
		t.Errorf("expected empty entries for missing task, got %d", len(m.Entries))
	}
}

func TestLoadLinkedActivitiesNoRefs(t *testing.T) {
	s := newFutureStore(t)
	now := time.Now().UTC()
	task := model.Task{ID: "f-001", Title: "No refs", Status: model.StatusTodo, CreatedAt: now, UpdatedAt: now}
	seedFutureTask(t, s, task)

	v := NewFutureView(s)
	cmd := v.loadLinkedActivities("f-001")
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	msg := cmd()
	m, ok := msg.(FutureLinkedActivitiesMsg)
	if !ok {
		t.Fatalf("expected FutureLinkedActivitiesMsg, got %T", msg)
	}
	if len(m.Entries) != 0 {
		t.Errorf("expected empty entries for task with no refs, got %d", len(m.Entries))
	}
}

func TestLoadLinkedActivitiesWithRefs(t *testing.T) {
	s := newFutureStore(t)
	now := time.Now().UTC()
	dateStr := now.Format(model.DateFormat)

	actEntry := model.ActivityEntry{
		ID:          "a-001",
		Timestamp:   now,
		Description: "did some work",
	}
	actLog := &model.ActivityLog{Date: now, Entries: []model.ActivityEntry{actEntry}}
	if err := s.SaveActivityLog(actLog); err != nil {
		t.Fatalf("SaveActivityLog: %v", err)
	}

	task := model.Task{
		ID:           "f-001",
		Title:        "Has refs",
		Status:       model.StatusTodo,
		CreatedAt:    now,
		UpdatedAt:    now,
		ActivityRefs: []model.ActivityRef{{ID: "a-001", Date: dateStr}},
	}
	seedFutureTask(t, s, task)

	v := NewFutureView(s)
	cmd := v.loadLinkedActivities("f-001")
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	msg := cmd()
	m, ok := msg.(FutureLinkedActivitiesMsg)
	if !ok {
		t.Fatalf("expected FutureLinkedActivitiesMsg, got %T", msg)
	}
	if len(m.Entries) != 1 || m.Entries[0].ID != "a-001" {
		t.Errorf("expected entry a-001, got %v", m.Entries)
	}
}

func TestLoadLinkedActivitiesGetActivitiesError(t *testing.T) {
	dir := t.TempDir()
	s, _ := storage.NewYAMLStore(dir)
	now := time.Now().UTC()

	// Task with no refs → GetActivitiesByRefs uses fallback date → reads activity file.
	// Write corrupt YAML so GetActivityLog returns an error.
	actPath := filepath.Join(dir, "activity", now.Format(model.DateFormat)+".yaml")
	if err := os.WriteFile(actPath, []byte(": invalid: yaml: \x00"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	task := model.Task{ID: "f-001", Title: "Err task", Status: model.StatusTodo, CreatedAt: now, UpdatedAt: now}
	seedFutureTask(t, s, task)

	v := NewFutureView(s)
	cmd := v.loadLinkedActivities("f-001")
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	msg := cmd()
	if _, ok := msg.(errMsg); !ok {
		t.Fatalf("expected errMsg, got %T", msg)
	}
}

// ── scheduleTask missing branches ─────────────────────────────────────────────

func TestScheduleTaskIDNotInList(t *testing.T) {
	s := newFutureStore(t)
	v := NewFutureView(s)

	cmd := v.scheduleTask("no-such-id", "today")
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	msg := cmd()
	if _, ok := msg.(FutureMsg); !ok {
		t.Fatalf("expected FutureMsg when task not found, got %T", msg)
	}
}

func TestScheduleTaskSaveFutureListFails(t *testing.T) {
	dir := t.TempDir()
	s, _ := storage.NewYAMLStore(dir)
	now := time.Now().UTC()
	task := model.Task{ID: "f-001", Title: "To schedule", Status: model.StatusTodo, CreatedAt: now, UpdatedAt: now}
	seedFutureTask(t, s, task)

	v := NewFutureView(s)

	// make future.yaml itself unwritable so SaveFutureList fails
	futureFile := filepath.Join(dir, "future.yaml")
	if err := os.Chmod(futureFile, 0o444); err != nil {
		t.Fatalf("Chmod: %v", err)
	}
	t.Cleanup(func() { os.Chmod(futureFile, 0o644) })

	cmd := v.scheduleTask("f-001", "today")
	msg := cmd()
	if _, ok := msg.(errMsg); !ok {
		t.Fatalf("expected errMsg when SaveFutureList fails, got %T", msg)
	}
}

// ── future loadMsg error branch ───────────────────────────────────────────────

func TestFutureLoadMsgError(t *testing.T) {
	dir := t.TempDir()
	s, _ := storage.NewYAMLStore(dir)

	futureFile := filepath.Join(dir, "future.yaml")
	if err := os.WriteFile(futureFile, []byte(": bad: yaml: \x00"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	v := NewFutureView(s)
	msg := v.loadMsg()
	if _, ok := msg.(errMsg); !ok {
		t.Fatalf("expected errMsg when GetFutureList fails, got %T", msg)
	}
}

// ── handleFutureAction missing branches ───────────────────────────────────────

func TestHandleFutureActionAddInDetailPane(t *testing.T) {
	s := newFutureStore(t)
	v := NewFutureView(s)
	v.pane = 1

	v2, cmd := v.handleFutureAction(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}, nil)
	if v2.adding {
		t.Error("'a' in detail pane should not open add form")
	}
	if cmd != nil {
		t.Error("cmd should be nil when 'a' pressed in detail pane")
	}
}

func TestHandleFutureActionEmptyList(t *testing.T) {
	s := newFutureStore(t)
	v := NewFutureView(s)

	v2, cmd := v.handleFutureAction(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}, nil)
	_ = v2
	if cmd != nil {
		t.Error("cmd should be nil when future task list is empty")
	}
}

// ── updateSchedule typing branch ─────────────────────────────────────────────

func TestUpdateScheduleTyping(t *testing.T) {
	v, _ := futureViewWithTask(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	v = v2

	v2, cmd := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	v = v2
	_ = cmd
	if v.scheduleInput.Value() == "" {
		t.Error("typing in schedule prompt should update the input value")
	}
}

// ── actions ───────────────────────────────────────────────────────────────────

func futureViewWithActions(t *testing.T) (FutureView, *storage.YAMLStore) {
	t.Helper()
	s := newFutureStore(t)
	now := time.Now().UTC()
	task := model.Task{
		ID: "f-20260525-001", Title: "Future task with actions", Status: model.StatusTodo,
		CreatedAt: now, UpdatedAt: now,
		Actions: []model.Action{
			{ID: "ac-001", Title: "future step one", Done: false},
			{ID: "ac-002", Title: "future step two", Done: true},
		},
	}
	seedFutureTask(t, s, task)
	v := NewFutureView(s, "work", "personal")
	list, _ := s.GetFutureList()
	v2, _ := v.Update(FutureMsg{List: list})
	return v2, s
}

func TestFutureViewOpenActions(t *testing.T) {
	v, _ := futureViewWithActions(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
	v = v2
	if v.detailMode != futureDetailActions {
		t.Fatalf("expected futureDetailActions, got %v", v.detailMode)
	}
	if v.pane != 1 {
		t.Error("pane should switch to 1 after 'A'")
	}
}

func TestFutureViewActionsEscExitsMode(t *testing.T) {
	v, _ := futureViewWithActions(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
	v = v2

	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyEsc})
	v = v2
	if v.detailMode != futureDetailNormal {
		t.Errorf("expected futureDetailNormal after esc, got %v", v.detailMode)
	}
}

func TestFutureViewActionsEscCancelsAdd(t *testing.T) {
	v, _ := futureViewWithActions(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
	v = v2
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	v = v2
	if !v.actionAdding {
		t.Fatal("expected actionAdding = true after 'a'")
	}

	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyEsc})
	v = v2
	if v.actionAdding {
		t.Error("actionAdding should be false after esc")
	}
	if v.detailMode != futureDetailActions {
		t.Errorf("detailMode should remain futureDetailActions, got %v", v.detailMode)
	}
}

func TestFutureViewActionsNavigate(t *testing.T) {
	v, _ := futureViewWithActions(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
	v = v2

	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	v = v2
	if v.actionCursor != 1 {
		t.Errorf("actionCursor = %d, want 1", v.actionCursor)
	}

	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	v = v2
	if v.actionCursor != 0 {
		t.Errorf("actionCursor = %d, want 0", v.actionCursor)
	}

	// clamp at 0
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	v = v2
	if v.actionCursor != 0 {
		t.Errorf("cursor should be clamped at 0, got %d", v.actionCursor)
	}
}

func TestFutureViewActionsAddEmptyNoOp(t *testing.T) {
	v, _ := futureViewWithActions(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
	v = v2
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	v = v2

	v2, cmd := v.Update(tea.KeyMsg{Type: tea.KeyEnter})
	v = v2
	if v.actionAdding {
		t.Error("actionAdding should be false after enter")
	}
	if cmd != nil {
		t.Error("empty title should produce nil cmd")
	}
}

func TestFutureViewActionsAddSave(t *testing.T) {
	v, s := futureViewWithActions(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
	v = v2
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	v = v2

	for _, r := range "third step" {
		v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		v = v2
	}

	_, cmd := v.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter with title should produce a save cmd")
	}
	msg := cmd()
	if _, ok := msg.(FutureMsg); !ok {
		t.Fatalf("expected FutureMsg, got %T", msg)
	}

	list, _ := s.GetFutureList()
	if len(list.Tasks[0].Actions) != 3 {
		t.Errorf("expected 3 actions, got %d", len(list.Tasks[0].Actions))
	}
	if list.Tasks[0].Actions[2].Title != "third step" {
		t.Errorf("new action title = %q, want 'third step'", list.Tasks[0].Actions[2].Title)
	}
}

func TestFutureViewActionsToggle(t *testing.T) {
	v, s := futureViewWithActions(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
	v = v2

	_, cmd := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	if cmd == nil {
		t.Fatal("space should produce a toggle cmd")
	}
	msg := cmd()
	if _, ok := msg.(FutureMsg); !ok {
		t.Fatalf("expected FutureMsg after toggle, got %T", msg)
	}

	list, _ := s.GetFutureList()
	if !list.Tasks[0].Actions[0].Done {
		t.Error("action 0 should be toggled to Done=true")
	}
}

func TestFutureViewActionsDelete(t *testing.T) {
	v, s := futureViewWithActions(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
	v = v2

	_, cmd := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'D'}})
	if cmd == nil {
		t.Fatal("D should produce a delete cmd")
	}
	msg := cmd()
	if _, ok := msg.(FutureMsg); !ok {
		t.Fatalf("expected FutureMsg after delete, got %T", msg)
	}

	list, _ := s.GetFutureList()
	if len(list.Tasks[0].Actions) != 1 {
		t.Errorf("expected 1 action after delete, got %d", len(list.Tasks[0].Actions))
	}
	if list.Tasks[0].Actions[0].ID != "ac-002" {
		t.Errorf("remaining action should be ac-002, got %s", list.Tasks[0].Actions[0].ID)
	}
}

func TestFutureViewActionsRenderDetail(t *testing.T) {
	v, _ := futureViewWithActions(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
	v = v2

	out := v.View(120, 40)
	if out == "" {
		t.Fatal("View should not be empty in futureDetailActions mode")
	}
	if !strings.Contains(out, "future step one") {
		t.Error("View should contain action title 'future step one'")
	}
}

func TestFutureViewActionsNoTaskNoOp(t *testing.T) {
	s := newFutureStore(t)
	v := NewFutureView(s)
	v.detailMode = futureDetailActions
	v2, cmd := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	_ = v2
	if cmd != nil {
		t.Error("space with no tasks should produce nil cmd")
	}
}
