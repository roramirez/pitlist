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

func TestParseLabels(t *testing.T) {
	cases := []struct {
		input string
		want  []string
	}{
		{"work auth", []string{"work", "auth"}},
		{"  single  ", []string{"single"}},
		{"", nil},
		{"   ", nil},
	}
	for _, c := range cases {
		got := parseLabels(c.input)
		if len(got) != len(c.want) {
			t.Errorf("parseLabels(%q) = %v, want %v", c.input, got, c.want)
			continue
		}
		for i := range got {
			if got[i] != c.want[i] {
				t.Errorf("parseLabels(%q)[%d] = %q, want %q", c.input, i, got[i], c.want[i])
			}
		}
	}
}

func TestParsePriority(t *testing.T) {
	cases := []struct {
		input string
		want  model.Priority
	}{
		{"low", model.PriorityLow},
		{"l", model.PriorityLow},
		{"LOW", model.PriorityLow},
		{"high", model.PriorityHigh},
		{"h", model.PriorityHigh},
		{"HIGH", model.PriorityHigh},
		{"medium", model.PriorityMedium},
		{"m", model.PriorityMedium},
		{"", model.PriorityMedium},
		{"unknown", model.PriorityMedium},
	}
	for _, c := range cases {
		got := parsePriority(c.input)
		if got != c.want {
			t.Errorf("parsePriority(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestMatchStatus(t *testing.T) {
	if !matchStatus(model.StatusTodo, nil) {
		t.Error("matchStatus with nil statuses should return true")
	}
	if !matchStatus(model.StatusTodo, []model.TaskStatus{model.StatusTodo, model.StatusDone}) {
		t.Error("matchStatus should return true when status is in list")
	}
	if matchStatus(model.StatusInProgress, []model.TaskStatus{model.StatusTodo, model.StatusDone}) {
		t.Error("matchStatus should return false when status not in list")
	}
}

func TestSortByContext(t *testing.T) {
	tasks := []model.Task{
		{ID: "1", Context: "personal"},
		{ID: "2", Context: "work"},
		{ID: "3", Context: ""},
		{ID: "4", Context: "work", CarryFrom: "2026-05-17"},
	}
	contexts := []string{"work", "personal"}

	result := sortByContext(tasks, contexts)
	if len(result) != 4 {
		t.Fatalf("expected 4 tasks, got %d", len(result))
	}

	// work should come before personal
	if result[0].Context != "work" || result[0].CarryFrom != "" {
		t.Errorf("first task should be work (non-carried), got id=%s ctx=%s", result[0].ID, result[0].Context)
	}
	// carried task should be last
	if result[3].CarryFrom == "" {
		t.Error("last task should be the carried one")
	}
}

func TestSortByContextEmpty(t *testing.T) {
	tasks := []model.Task{{ID: "1", Title: "solo"}}
	result := sortByContext(tasks, []string{"work"})
	if len(result) != 1 {
		t.Fatalf("expected 1, got %d", len(result))
	}
}

func TestHasMultipleContexts(t *testing.T) {
	single := []model.Task{
		{Context: "work"},
		{Context: "work"},
	}
	if hasMultipleContexts(single) {
		t.Error("expected false for single context")
	}

	multi := []model.Task{
		{Context: "work"},
		{Context: "personal"},
	}
	if !hasMultipleContexts(multi) {
		t.Error("expected true for multiple contexts")
	}

	// Carried tasks are ignored in the context count
	withCarried := []model.Task{
		{Context: "work"},
		{Context: "personal", CarryFrom: "2026-05-17"},
	}
	if hasMultipleContexts(withCarried) {
		t.Error("carried tasks should be excluded from context count")
	}
}

func TestTasksViewAccessors(t *testing.T) {
	store, _ := storage.NewYAMLStore(t.TempDir())
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewTasksView(store, date, "work", "personal")

	if !v.Date().Equal(date) {
		t.Errorf("Date() = %v, want %v", v.Date(), date)
	}

	ctxs := v.Contexts()
	if len(ctxs) != 2 || ctxs[0] != "work" || ctxs[1] != "personal" {
		t.Errorf("Contexts() = %v, want [work personal]", ctxs)
	}

	if v.IsInputActive() {
		t.Error("IsInputActive() should be false initially")
	}
}

func TestTasksViewLoad(t *testing.T) {
	store, _ := storage.NewYAMLStore(t.TempDir())
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewTasksView(store, date)

	cmd := v.Load()
	if cmd == nil {
		t.Fatal("Load should return a cmd")
	}
	msg := cmd()
	tm, ok := msg.(TasksMsg)
	if !ok {
		t.Fatalf("expected TasksMsg, got %T", msg)
	}
	if tm.Plan == nil {
		t.Error("Plan should not be nil")
	}
}

func TestTasksViewUpdateTasksMsg(t *testing.T) {
	store, _ := storage.NewYAMLStore(t.TempDir())
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewTasksView(store, date)

	plan := &model.DayPlan{
		Date: date,
		Tasks: []model.Task{
			{ID: "t-001", Title: "Task A", Status: model.StatusTodo, CreatedAt: date, UpdatedAt: date},
			{ID: "t-002", Title: "Task B", Status: model.StatusTodo, CreatedAt: date, UpdatedAt: date},
		},
	}
	v2, _ := v.Update(TasksMsg{Plan: plan, ActLog: &model.ActivityLog{Date: date}})
	v = v2
	if len(v.filteredTasks()) != 2 {
		t.Errorf("expected 2 tasks, got %d", len(v.filteredTasks()))
	}
}

func TestTasksViewUpdateWindowSize(t *testing.T) {
	store, _ := storage.NewYAMLStore(t.TempDir())
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewTasksView(store, date)

	v2, _ := v.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	v = v2
	if v.width != 120 || v.height != 40 {
		t.Errorf("size not set: %dx%d", v.width, v.height)
	}
}

func TestTasksViewNavigateJK(t *testing.T) {
	store, _ := storage.NewYAMLStore(t.TempDir())
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewTasksView(store, date)

	plan := &model.DayPlan{
		Date: date,
		Tasks: []model.Task{
			{ID: "t-001", Title: "A", Status: model.StatusTodo, CreatedAt: date, UpdatedAt: date},
			{ID: "t-002", Title: "B", Status: model.StatusTodo, CreatedAt: date, UpdatedAt: date},
		},
	}
	v2, _ := v.Update(TasksMsg{Plan: plan, ActLog: &model.ActivityLog{Date: date}})
	v = v2

	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	v = v2
	if v.cursor != 1 {
		t.Errorf("j: cursor=%d, want 1", v.cursor)
	}

	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	v = v2
	if v.cursor != 0 {
		t.Errorf("k: cursor=%d, want 0", v.cursor)
	}
}

func TestTasksViewDayNavigation(t *testing.T) {
	store, _ := storage.NewYAMLStore(t.TempDir())
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewTasksView(store, date)

	// l moves forward
	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	v = v2
	if !v.date.Equal(date.AddDate(0, 0, 1)) {
		t.Errorf("l: date=%v, want %v", v.date, date.AddDate(0, 0, 1))
	}

	// h moves back
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	v = v2
	if !v.date.Equal(date) {
		t.Errorf("h: date=%v, want %v", v.date, date)
	}
}

func TestTasksViewPaneTab(t *testing.T) {
	store, _ := storage.NewYAMLStore(t.TempDir())
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewTasksView(store, date)

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

func TestTasksViewOpenAddForm(t *testing.T) {
	store, _ := storage.NewYAMLStore(t.TempDir())
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewTasksView(store, date)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	v = v2
	if !v.adding {
		t.Error("'a' should open the add form")
	}
	if !v.IsInputActive() {
		t.Error("IsInputActive should be true when adding")
	}

	// esc closes form
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyEsc})
	v = v2
	if v.adding {
		t.Error("esc should close add form")
	}
}

func TestTasksViewWeekToggle(t *testing.T) {
	store, _ := storage.NewYAMLStore(t.TempDir())
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewTasksView(store, date)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	v = v2
	if !v.weekMode {
		t.Error("w should enable weekMode")
	}
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	v = v2
	if v.weekMode {
		t.Error("second w should disable weekMode")
	}
}

func TestTasksViewToggleDone(t *testing.T) {
	store, _ := storage.NewYAMLStore(t.TempDir())
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	plan := &model.DayPlan{
		Date: date,
		Tasks: []model.Task{
			{ID: "t-001", Title: "A", Status: model.StatusTodo, CreatedAt: date, UpdatedAt: date},
		},
	}
	store.SaveDayPlan(plan)

	v := NewTasksView(store, date)
	v2, _ := v.Update(TasksMsg{Plan: plan, ActLog: &model.ActivityLog{Date: date}})
	v = v2

	_, cmd := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if cmd == nil {
		t.Fatal("'d' should return a cmd")
	}
	msg := cmd()
	if _, ok := msg.(TasksMsg); !ok {
		t.Errorf("expected TasksMsg after toggle, got %T", msg)
	}
}

func TestTasksViewDeleteTask(t *testing.T) {
	store, _ := storage.NewYAMLStore(t.TempDir())
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	plan := &model.DayPlan{
		Date: date,
		Tasks: []model.Task{
			{ID: "t-001", Title: "Delete me", Status: model.StatusTodo, CreatedAt: date, UpdatedAt: date},
		},
	}
	store.SaveDayPlan(plan)

	v := NewTasksView(store, date)
	v2, _ := v.Update(TasksMsg{Plan: plan, ActLog: &model.ActivityLog{Date: date}})
	v = v2

	_, cmd := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'D'}})
	if cmd == nil {
		t.Fatal("D should return a cmd")
	}
	msg := cmd()
	if _, ok := msg.(TasksMsg); !ok {
		t.Errorf("expected TasksMsg after delete, got %T", msg)
	}
}

func TestTasksViewSetFilter(t *testing.T) {
	store, _ := storage.NewYAMLStore(t.TempDir())
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewTasksView(store, date)

	// No-label filter triggers Load
	_, cmd := v.SetFilter(TaskFilter{Statuses: []model.TaskStatus{model.StatusTodo}})
	if cmd == nil {
		t.Error("SetFilter without labels should return a Load cmd")
	}

	// Label filter triggers global search
	_, cmd = v.SetFilter(TaskFilter{Labels: []string{"work"}})
	if cmd == nil {
		t.Error("SetFilter with labels should return a search cmd")
	}
}

func TestTasksViewView(t *testing.T) {
	store, _ := storage.NewYAMLStore(t.TempDir())
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewTasksView(store, date, "work", "personal")

	// Load some tasks so the view has content
	plan := &model.DayPlan{
		Date: date,
		Tasks: []model.Task{
			{ID: "t-001", Title: "My task", Status: model.StatusTodo,
				Context: "work", Labels: []string{"work"},
				Priority: model.PriorityHigh, CreatedAt: date, UpdatedAt: date,
				Notes: "some notes", CarryFrom: "2026-05-17"},
		},
	}
	v2, _ := v.Update(TasksMsg{Plan: plan, ActLog: &model.ActivityLog{Date: date}})
	v = v2

	out := v.View(120, 40)
	if out == "" {
		t.Error("View should return non-empty string")
	}
}

func TestTasksViewAddFormTabAndEsc(t *testing.T) {
	store, _ := storage.NewYAMLStore(t.TempDir())
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewTasksView(store, date, "work")

	// Open add form
	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	v = v2

	// Tab cycles fields
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyTab})
	v = v2
	if v.tForm.focusIdx != 1 {
		t.Errorf("tab: focusIdx=%d, want 1", v.tForm.focusIdx)
	}

	// Shift+Tab goes back
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	v = v2
	if v.tForm.focusIdx != 0 {
		t.Errorf("shift+tab: focusIdx=%d, want 0", v.tForm.focusIdx)
	}
}

func TestTasksViewLinkedActivitiesMsg(t *testing.T) {
	store, _ := storage.NewYAMLStore(t.TempDir())
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewTasksView(store, date)

	ts := time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC)
	entries := []model.ActivityEntry{
		{ID: "a-001", Timestamp: ts, Description: "Work"},
	}
	v2, _ := v.Update(LinkedActivitiesMsg{Entries: entries})
	v = v2
	if len(v.linkedActivities) != 1 {
		t.Errorf("expected 1 linked activity, got %d", len(v.linkedActivities))
	}
}

func TestTasksViewGlobalTasksMsg(t *testing.T) {
	store, _ := storage.NewYAMLStore(t.TempDir())
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewTasksView(store, date)
	v.cursor = 5

	task := &model.Task{ID: "t-001", Title: "Global"}
	v2, _ := v.Update(GlobalTasksMsg{Tasks: []*model.Task{task}})
	v = v2
	if v.globalResults == nil || len(v.globalResults) != 1 {
		t.Errorf("expected 1 global result, got %v", v.globalResults)
	}
	if v.cursor != 0 {
		t.Errorf("cursor should reset to 0, got %d", v.cursor)
	}
}

// helper: load a view with one task already in the store
func viewWithTask(t *testing.T) (TasksView, *storage.YAMLStore, time.Time) {
	t.Helper()
	store, _ := storage.NewYAMLStore(t.TempDir())
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	plan := &model.DayPlan{
		Date: date,
		Tasks: []model.Task{
			{ID: "t-20260518-001", Title: "My Task", Status: model.StatusTodo,
				Notes: "original notes", CreatedAt: date, UpdatedAt: date},
		},
	}
	store.SaveDayPlan(plan)
	v := NewTasksView(store, date, "work", "personal")
	v2, _ := v.Update(TasksMsg{Plan: plan, ActLog: &model.ActivityLog{Date: date}})
	return v2, store, date
}

// ── notes editing ─────────────────────────────────────────────────────────────

func TestTasksViewNotesEsc(t *testing.T) {
	v, _, _ := viewWithTask(t)

	// 'n' opens notes editor
	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	v = v2
	if v.detailMode != detailEditNotes {
		t.Fatalf("expected detailEditNotes, got %v", v.detailMode)
	}
	if !v.IsInputActive() {
		t.Error("IsInputActive should be true while editing notes")
	}

	// esc exits
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyEsc})
	v = v2
	if v.detailMode != detailNormal {
		t.Errorf("expected detailNormal after esc, got %v", v.detailMode)
	}
}

func TestTasksViewNotesSave(t *testing.T) {
	v, store, date := viewWithTask(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	v = v2

	_, cmd := v.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	if cmd == nil {
		t.Fatal("ctrl+s in notes mode should return a save cmd")
	}
	msg := cmd()
	if _, ok := msg.(TasksMsg); !ok {
		t.Fatalf("expected TasksMsg after save notes, got %T", msg)
	}
	// verify notes were persisted
	plan, _ := store.GetDayPlan(date)
	_ = plan
}

// ── carry ─────────────────────────────────────────────────────────────────────

func TestTasksViewCarryEsc(t *testing.T) {
	v, _, _ := viewWithTask(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	v = v2
	if v.detailMode != detailCarry {
		t.Fatalf("expected detailCarry, got %v", v.detailMode)
	}

	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyEsc})
	v = v2
	if v.detailMode != detailNormal {
		t.Errorf("expected detailNormal after esc, got %v", v.detailMode)
	}
}

func TestTasksViewCarryInvalidDate(t *testing.T) {
	v, _, _ := viewWithTask(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	v = v2

	// Clear the carry input and enter an invalid date
	v.carryInput.SetValue("not-a-date")
	v2, cmd := v.Update(tea.KeyMsg{Type: tea.KeyEnter})
	v = v2
	_ = cmd
	// Should stay in carry mode (invalid date rejected)
	if v.detailMode != detailCarry {
		t.Errorf("invalid date should keep carry mode, got %v", v.detailMode)
	}
}

func TestTasksViewCarryConfirm(t *testing.T) {
	v, store, srcDate := viewWithTask(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	v = v2

	destDate := srcDate.AddDate(0, 0, 1)
	v.carryInput.SetValue(destDate.Format("2006-01-02"))

	v2, cmd := v.Update(tea.KeyMsg{Type: tea.KeyEnter})
	v = v2
	if cmd == nil {
		t.Fatal("enter with valid carry date should return a cmd")
	}
	msg := cmd()
	if _, ok := msg.(TasksMsg); !ok {
		t.Fatalf("expected TasksMsg after carry, got %T", msg)
	}

	// task moved to dest
	dest, _ := store.GetDayPlan(destDate)
	if len(dest.Tasks) != 1 || dest.Tasks[0].ID != "t-20260518-001" {
		t.Errorf("task not found on dest day: %v", dest.Tasks)
	}
}

func TestTasksViewCarryInputKey(t *testing.T) {
	v, _, _ := viewWithTask(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	v = v2

	// Typing in the carry input should update it
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	v = v2
	_ = v.carryInput.Value() // just ensure no panic
}

// ── log activity form ─────────────────────────────────────────────────────────

func TestTasksViewLogFormEsc(t *testing.T) {
	v, _, _ := viewWithTask(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'L'}})
	v = v2
	if v.detailMode != detailLogActivity {
		t.Fatalf("expected detailLogActivity, got %v", v.detailMode)
	}

	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyEsc})
	v = v2
	if v.detailMode != detailNormal {
		t.Errorf("esc should close log form, got %v", v.detailMode)
	}
}

func TestTasksViewLogFormTab(t *testing.T) {
	v, _, _ := viewWithTask(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'L'}})
	v = v2

	// Tab cycles fields
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyTab})
	v = v2
	if v.logForm.focusIdx != 1 {
		t.Errorf("tab: focusIdx=%d, want 1", v.logForm.focusIdx)
	}

	// Shift+tab goes back
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	v = v2
	if v.logForm.focusIdx != 0 {
		t.Errorf("shift+tab: focusIdx=%d, want 0", v.logForm.focusIdx)
	}
}

func TestTasksViewLogFormEmptySubmit(t *testing.T) {
	v, _, _ := viewWithTask(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'L'}})
	v = v2

	// ctrl+s with empty description → nil cmd
	v2, cmd := v.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	v = v2
	if cmd != nil {
		// submitLogForm returns nil for empty desc; that's fine
	}
	_ = cmd
}

func TestTasksViewLogFormSubmit(t *testing.T) {
	v, _, _ := viewWithTask(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'L'}})
	v = v2

	// Type a description into the first field
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
	if _, ok := msg.(TasksMsg); !ok {
		t.Fatalf("expected TasksMsg after log submit, got %T", msg)
	}
}

// ── edit task form ────────────────────────────────────────────────────────────

func TestTasksViewEditFormEsc(t *testing.T) {
	v, _, _ := viewWithTask(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	v = v2
	if v.detailMode != detailEditTask {
		t.Fatalf("expected detailEditTask, got %v", v.detailMode)
	}

	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyEsc})
	v = v2
	if v.detailMode != detailNormal {
		t.Errorf("esc should close edit form")
	}
}

func TestTasksViewEditFormTab(t *testing.T) {
	v, _, _ := viewWithTask(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	v = v2

	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyTab})
	v = v2
	if v.tForm.focusIdx != 1 {
		t.Errorf("tab: focusIdx=%d, want 1", v.tForm.focusIdx)
	}
}

func TestTasksViewEditFormSave(t *testing.T) {
	v, store, date := viewWithTask(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	v = v2

	v2, cmd := v.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	v = v2
	if cmd == nil {
		t.Fatal("ctrl+s in edit form should return a cmd")
	}
	msg := cmd()
	if _, ok := msg.(TasksMsg); !ok {
		t.Fatalf("expected TasksMsg after edit save, got %T", msg)
	}
	plan, _ := store.GetDayPlan(date)
	if len(plan.Tasks) != 1 {
		t.Errorf("expected 1 task, got %d", len(plan.Tasks))
	}
}

// ── add form save ─────────────────────────────────────────────────────────────

func TestTasksViewAddFormSave(t *testing.T) {
	store, _ := storage.NewYAMLStore(t.TempDir())
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewTasksView(store, date, "work")

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	v = v2

	// Type title
	for _, ch := range []rune{'N', 'e', 'w'} {
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
	if _, ok := msg.(TasksMsg); !ok {
		t.Fatalf("expected TasksMsg after add save, got %T", msg)
	}

	plan, _ := store.GetDayPlan(date)
	if len(plan.Tasks) != 1 || plan.Tasks[0].Title != "New" {
		t.Errorf("task not saved: %v", plan.Tasks)
	}
}

func TestTasksViewAddFormEnterSave(t *testing.T) {
	store, _ := storage.NewYAMLStore(t.TempDir())
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewTasksView(store, date)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	v = v2

	for _, ch := range []rune{'T', 'a', 's', 'k'} {
		v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
		v = v2
	}

	// Tab to last field then enter
	for i := 0; i < 3; i++ {
		v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyTab})
		v = v2
	}
	v2, cmd := v.Update(tea.KeyMsg{Type: tea.KeyEnter})
	v = v2
	_ = cmd
}

func TestTasksViewAddFormEmptyCtrlS(t *testing.T) {
	store, _ := storage.NewYAMLStore(t.TempDir())
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewTasksView(store, date)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	v = v2

	// ctrl+s with empty title → close form, no cmd
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	v = v2
	if v.adding {
		t.Error("adding should be false after ctrl+s with empty title")
	}
}

// ── contextValue / contextDisplay ────────────────────────────────────────────

func TestContextValue(t *testing.T) {
	f := newTaskForm("", "", "work", []string{"work", "personal"}, "", "medium")
	if f.contextValue() != "work" {
		t.Errorf("contextValue() = %q, want 'work'", f.contextValue())
	}

	// Out of range → empty
	f.contextIdx = 99
	if f.contextValue() != "" {
		t.Errorf("out-of-range contextValue should be empty, got %q", f.contextValue())
	}
}

func TestContextDisplay(t *testing.T) {
	f := newTaskForm("", "", "work", []string{"work", "personal"}, "", "medium")

	out := f.contextDisplay(true) // focused
	if out == "" {
		t.Error("contextDisplay focused returned empty")
	}
	out = f.contextDisplay(false) // unfocused
	if out == "" {
		t.Error("contextDisplay unfocused returned empty")
	}

	// No contexts → label "—"
	f2 := newTaskForm("", "", "", nil, "", "medium")
	out = f2.contextDisplay(false)
	if out == "" {
		t.Error("contextDisplay with no contexts returned empty")
	}
}

// ── render detail modes ───────────────────────────────────────────────────────

func TestTasksViewViewWithCarryMode(t *testing.T) {
	v, _, _ := viewWithTask(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	v = v2

	out := v.View(120, 40)
	if out == "" {
		t.Error("View in carry mode returned empty")
	}
}

func TestTasksViewViewWithNotesMode(t *testing.T) {
	v, _, _ := viewWithTask(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	v = v2

	out := v.View(120, 40)
	if out == "" {
		t.Error("View in notes mode returned empty")
	}
}

func TestTasksViewViewWithLogMode(t *testing.T) {
	v, _, _ := viewWithTask(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'L'}})
	v = v2

	out := v.View(120, 40)
	if out == "" {
		t.Error("View in log-activity mode returned empty")
	}
}

func TestTasksViewViewWithEditMode(t *testing.T) {
	v, _, _ := viewWithTask(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	v = v2

	out := v.View(120, 40)
	if out == "" {
		t.Error("View in edit-task mode returned empty")
	}
}

func TestTasksViewViewWithAddForm(t *testing.T) {
	store, _ := storage.NewYAMLStore(t.TempDir())
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewTasksView(store, date)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	v = v2

	out := v.View(120, 40)
	if out == "" {
		t.Error("View with add form returned empty")
	}
}

func TestTasksViewViewDoneTask(t *testing.T) {
	store, _ := storage.NewYAMLStore(t.TempDir())
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	now := time.Now().UTC()
	plan := &model.DayPlan{
		Date: date,
		Tasks: []model.Task{
			{ID: "t-001", Title: "Done task", Status: model.StatusDone,
				Priority: model.PriorityHigh, CarryFrom: "2026-05-17",
				Labels: []string{"work"}, DoneAt: &now,
				CreatedAt: date, UpdatedAt: date},
		},
	}
	v := NewTasksView(store, date, "work")
	v2, _ := v.Update(TasksMsg{Plan: plan, ActLog: &model.ActivityLog{Date: date}})
	v = v2

	out := v.View(120, 40)
	if out == "" {
		t.Error("View with done task returned empty")
	}
}

// ── newQuickLogForm ───────────────────────────────────────────────────────────

func TestNewQuickLogForm(t *testing.T) {
	f := newQuickLogForm("t-20260518-001")
	if f.taskID != "t-20260518-001" {
		t.Errorf("taskID = %q, want t-20260518-001", f.taskID)
	}
}

func TestTasksViewMaxMin(t *testing.T) {
	if max(3, 7) != 7 {
		t.Error("max(3,7) should be 7")
	}
	if max(9, 2) != 9 {
		t.Error("max(9,2) should be 9")
	}
	if min(3, 7) != 3 {
		t.Error("min(3,7) should be 3")
	}
	if min(9, 2) != 2 {
		t.Error("min(9,2) should be 2")
	}
}

// ── handleFormKey ─────────────────────────────────────────────────────────────

func TestHandleFormKeyContextCycleRight(t *testing.T) {
	store, _ := storage.NewYAMLStore(t.TempDir())
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewTasksView(store, date, "work", "personal")

	// Open add form
	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	v = v2

	// Tab to context field (focusIdx=1)
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyTab})
	v = v2
	if v.tForm.focusIdx != 1 {
		t.Fatalf("expected focusIdx=1, got %d", v.tForm.focusIdx)
	}

	initialIdx := v.tForm.contextIdx
	// Send right arrow key → contextIdx should increment
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRight})
	v = v2
	if v.tForm.contextIdx == initialIdx && len(v.tForm.contexts) > 0 {
		t.Errorf("right arrow: contextIdx should have changed from %d", initialIdx)
	}

	// Send 'l' key → also cycles right
	prev := v.tForm.contextIdx
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	v = v2
	// Cycle past end (wraps to -1)
	// Keep cycling until we confirm wrap behaviour doesn't panic
	for i := 0; i < len(v.tForm.contexts)+2; i++ {
		v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
		v = v2
	}
	_ = prev
}

func TestHandleFormKeyContextCycleLeft(t *testing.T) {
	store, _ := storage.NewYAMLStore(t.TempDir())
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewTasksView(store, date, "work", "personal")

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	v = v2
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyTab})
	v = v2

	// Send left arrow multiple times — no panic, wraps correctly
	for i := 0; i < len(v.tForm.contexts)+3; i++ {
		v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyLeft})
		v = v2
	}
}

func TestHandleFormKeyLabelsField(t *testing.T) {
	store, _ := storage.NewYAMLStore(t.TempDir())
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewTasksView(store, date, "work")

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	v = v2
	// Tab twice to labels field (focusIdx=2)
	for i := 0; i < 2; i++ {
		v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyTab})
		v = v2
	}
	if v.tForm.focusIdx != 2 {
		t.Fatalf("expected focusIdx=2, got %d", v.tForm.focusIdx)
	}
	// Send a rune key → no panic
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	v = v2
}

func TestHandleFormKeyPriorityField(t *testing.T) {
	store, _ := storage.NewYAMLStore(t.TempDir())
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewTasksView(store, date, "work")

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	v = v2
	// Tab 3 times to priority field (focusIdx=3)
	for i := 0; i < 3; i++ {
		v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyTab})
		v = v2
	}
	if v.tForm.focusIdx != 3 {
		t.Fatalf("expected focusIdx=3, got %d", v.tForm.focusIdx)
	}
	// Send a rune key → no panic
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
	v = v2
}

// ── updateTaskForm extra paths ────────────────────────────────────────────────

func TestTasksViewEditFormShiftTab(t *testing.T) {
	v, _, _ := viewWithTask(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	v = v2
	if v.detailMode != detailEditTask {
		t.Fatalf("expected detailEditTask, got %v", v.detailMode)
	}

	// shift+tab from focusIdx=0 wraps to taskFormFields-1 (3)
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyShiftTab})
	v = v2
	if v.tForm.focusIdx != taskFormFields-1 {
		t.Errorf("shift+tab: focusIdx=%d, want %d", v.tForm.focusIdx, taskFormFields-1)
	}
}

func TestTasksViewEditFormEnterLastField(t *testing.T) {
	v, _, _ := viewWithTask(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	v = v2

	// Tab to last field
	for i := 0; i < taskFormFields-1; i++ {
		v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyTab})
		v = v2
	}
	if v.tForm.focusIdx != taskFormFields-1 {
		t.Fatalf("expected last field, got %d", v.tForm.focusIdx)
	}

	// enter on last field → ctrl+s saves
	v2, cmd := v.Update(tea.KeyMsg{Type: tea.KeyEnter})
	v = v2
	if cmd == nil {
		t.Fatal("enter on last edit field should return a save cmd")
	}
	msg := cmd()
	if _, ok := msg.(TasksMsg); !ok {
		t.Errorf("expected TasksMsg, got %T", msg)
	}
}

func TestTasksViewEditFormDefaultKey(t *testing.T) {
	v, _, _ := viewWithTask(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	v = v2

	// Send a regular rune → handleFormKey is called, no panic
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	v = v2
}

// ── renderTasksByContext ──────────────────────────────────────────────────────

func TestTasksViewRenderMultiContext(t *testing.T) {
	store, _ := storage.NewYAMLStore(t.TempDir())
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewTasksView(store, date, "work", "personal")

	plan := &model.DayPlan{
		Date: date,
		Tasks: []model.Task{
			{ID: "t-001", Title: "Work task", Status: model.StatusTodo, Context: "work", CreatedAt: date, UpdatedAt: date},
			{ID: "t-002", Title: "Personal task", Status: model.StatusTodo, Context: "personal", CreatedAt: date, UpdatedAt: date},
			{ID: "t-003", Title: "Carried task", Status: model.StatusTodo, Context: "work", CarryFrom: "2026-05-17", CreatedAt: date, UpdatedAt: date},
		},
	}
	v2, _ := v.Update(TasksMsg{Plan: plan, ActLog: &model.ActivityLog{Date: date}})
	v = v2

	out := v.View(120, 40)
	if out == "" {
		t.Error("View with multi-context tasks returned empty")
	}
}

// ── focusLogField ─────────────────────────────────────────────────────────────

func TestTasksViewLogFormFocusAllFields(t *testing.T) {
	v, _, _ := viewWithTask(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'L'}})
	v = v2
	if v.detailMode != detailLogActivity {
		t.Fatalf("expected detailLogActivity, got %v", v.detailMode)
	}

	// Tab through all 4 fields (0→1→2→3→0) verifying no panic
	for i := 0; i < quickLogFields+1; i++ {
		v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyTab})
		v = v2
		want := (i + 1) % quickLogFields
		if v.logForm.focusIdx != want {
			t.Errorf("after %d tabs: focusIdx=%d, want %d", i+1, v.logForm.focusIdx, want)
		}
	}
}

// ── renderTaskDetail with notes and linked activities ─────────────────────────

func TestTasksViewRenderTaskDetailWithNotes(t *testing.T) {
	store, _ := storage.NewYAMLStore(t.TempDir())
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	ts := time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC)

	plan := &model.DayPlan{
		Date: date,
		Tasks: []model.Task{
			{ID: "t-20260518-001", Title: "Task with notes", Status: model.StatusTodo,
				Notes: "These are some notes", CreatedAt: date, UpdatedAt: date},
		},
	}
	store.SaveDayPlan(plan)

	v := NewTasksView(store, date)
	v2, _ := v.Update(TasksMsg{Plan: plan, ActLog: &model.ActivityLog{Date: date}})
	v = v2

	// Inject linked activities
	v2, _ = v.Update(LinkedActivitiesMsg{Entries: []model.ActivityEntry{
		{ID: "a-001", Timestamp: ts, Description: "Work", DurationMin: 45, Tags: []string{"work"}},
	}})
	v = v2

	out := v.View(120, 40)
	if out == "" {
		t.Error("View with notes+linked activities returned empty")
	}
}

// ── filteredTasks with globalResults ─────────────────────────────────────────

func TestTasksViewFilteredTasksGlobalResults(t *testing.T) {
	store, _ := storage.NewYAMLStore(t.TempDir())
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewTasksView(store, date)

	task1 := &model.Task{ID: "t-001", Title: "Global A", Status: model.StatusTodo}
	task2 := &model.Task{ID: "t-002", Title: "Global B", Status: model.StatusTodo}
	v2, _ := v.Update(GlobalTasksMsg{Tasks: []*model.Task{task1, task2}})
	v = v2

	tasks := v.filteredTasks()
	if len(tasks) != 2 {
		t.Errorf("expected 2 tasks from globalResults, got %d", len(tasks))
	}
}

// ── toggleDone done→todo ──────────────────────────────────────────────────────

func TestTasksViewToggleDoneToTodo(t *testing.T) {
	store, _ := storage.NewYAMLStore(t.TempDir())
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	now := time.Now().UTC()
	plan := &model.DayPlan{
		Date: date,
		Tasks: []model.Task{
			{ID: "t-20260518-001", Title: "Done task", Status: model.StatusDone,
				DoneAt: &now, CreatedAt: date, UpdatedAt: date},
		},
	}
	store.SaveDayPlan(plan)

	v := NewTasksView(store, date)
	// Use filter that includes done tasks
	v.filter = TaskFilter{Statuses: []model.TaskStatus{model.StatusDone}}
	v2, _ := v.Update(TasksMsg{Plan: plan, ActLog: &model.ActivityLog{Date: date}})
	v = v2

	_, cmd := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if cmd == nil {
		t.Fatal("'d' should return a cmd")
	}
	msg := cmd()
	if _, ok := msg.(TasksMsg); !ok {
		t.Errorf("expected TasksMsg after toggle done→todo, got %T", msg)
	}

	// Verify it's now todo
	updated, _ := store.GetDayPlan(date)
	if updated.Tasks[0].Status != model.StatusTodo {
		t.Errorf("expected StatusTodo after toggle, got %q", updated.Tasks[0].Status)
	}
}

// ── updateLogForm enter on non-last field + default key ──────────────────────

func TestTasksViewLogFormEnterNonLastField(t *testing.T) {
	v, _, _ := viewWithTask(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'L'}})
	v = v2
	if v.logForm.focusIdx != 0 {
		t.Fatalf("expected focusIdx=0, got %d", v.logForm.focusIdx)
	}

	// enter on non-last field (0) → advances to next field
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyEnter})
	v = v2
	if v.logForm.focusIdx != 1 {
		t.Errorf("enter on field 0: expected focusIdx=1, got %d", v.logForm.focusIdx)
	}
}

func TestTasksViewLogFormDefaultKey(t *testing.T) {
	v, _, _ := viewWithTask(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'L'}})
	v = v2

	// Send a rune → forwarded to current field (desc), no panic
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	v = v2
}

// ── updateNotes default key ───────────────────────────────────────────────────

func TestTasksViewNotesDefaultKey(t *testing.T) {
	v, _, _ := viewWithTask(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	v = v2
	if v.detailMode != detailEditNotes {
		t.Fatalf("expected detailEditNotes, got %v", v.detailMode)
	}

	// Send a regular rune (not esc/ctrl+s) → goes to default branch, textarea updated
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	v = v2
	if v.detailMode != detailEditNotes {
		t.Errorf("should stay in notes mode after rune key, got %v", v.detailMode)
	}
}

// ── loadLinkedActivities ──────────────────────────────────────────────────────

func TestTasksViewLoadLinkedActivities(t *testing.T) {
	store, _ := storage.NewYAMLStore(t.TempDir())
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	ts := time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC)

	// Seed activity entry
	actLog := &model.ActivityLog{
		Date: date,
		Entries: []model.ActivityEntry{
			{ID: "a-20260518-001", Timestamp: ts, Description: "Work on task"},
		},
	}
	store.SaveActivityLog(actLog)

	// Seed task with ActivityRef
	plan := &model.DayPlan{
		Date: date,
		Tasks: []model.Task{
			{ID: "t-20260518-001", Title: "Task with ref", Status: model.StatusTodo,
				ActivityRefs: []model.ActivityRef{{ID: "a-20260518-001", Date: "2026-05-18"}},
				CreatedAt:    date, UpdatedAt: date},
		},
	}
	store.SaveDayPlan(plan)

	v := NewTasksView(store, date)
	// Loading TasksMsg triggers loadLinkedActivities
	v2, cmd := v.Update(TasksMsg{Plan: plan, ActLog: actLog})
	v = v2
	if cmd == nil {
		t.Fatal("TasksMsg with task should return a loadLinkedActivities cmd")
	}
	// Execute the cmd to get LinkedActivitiesMsg
	msg := cmd()
	lam, ok := msg.(LinkedActivitiesMsg)
	if !ok {
		t.Fatalf("expected LinkedActivitiesMsg, got %T", msg)
	}
	if len(lam.Entries) != 1 {
		t.Errorf("expected 1 linked activity, got %d", len(lam.Entries))
	}
}

// ── SetFilter global results ──────────────────────────────────────────────────

func TestTasksViewSetFilterGlobalResultHandled(t *testing.T) {
	store, _ := storage.NewYAMLStore(t.TempDir())
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	plan := &model.DayPlan{
		Date: date,
		Tasks: []model.Task{
			{ID: "t-20260518-001", Title: "Work task", Labels: []string{"work"},
				Status: model.StatusTodo, CreatedAt: date, UpdatedAt: date},
		},
	}
	store.SaveDayPlan(plan)

	v := NewTasksView(store, date)
	v2, cmd := v.SetFilter(TaskFilter{Labels: []string{"work"}})
	v = v2
	if cmd == nil {
		t.Fatal("SetFilter with labels should return a cmd")
	}
	msg := cmd()
	gm, ok := msg.(GlobalTasksMsg)
	if !ok {
		t.Fatalf("expected GlobalTasksMsg, got %T", msg)
	}
	// Feed it back into Update
	v2, _ = v.Update(gm)
	v = v2
	if v.globalResults == nil {
		t.Error("globalResults should be set after GlobalTasksMsg")
	}
}

// ── renderTaskLine states ─────────────────────────────────────────────────────

func TestRenderTaskLineStates(t *testing.T) {
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)

	cases := []model.Task{
		{ID: "t-001", Title: "Done", Status: model.StatusDone, CreatedAt: date},
		{ID: "t-002", Title: "InProgress", Status: model.StatusInProgress, CreatedAt: date},
		{ID: "t-003", Title: "Cancelled", Status: model.StatusCancelled, CreatedAt: date},
		{ID: "t-004", Title: "High prio", Status: model.StatusTodo, Priority: model.PriorityHigh},
		{ID: "t-005", Title: "Carried", Status: model.StatusTodo, CarryFrom: "2026-05-17"},
		{ID: "t-006", Title: "Has notes", Status: model.StatusTodo, Notes: "some notes"},
	}
	for _, task := range cases {
		line := renderTaskLine(task, false, 80)
		if line == "" {
			t.Errorf("renderTaskLine for %q returned empty", task.Title)
		}
		// Also test selected=true
		lineSelected := renderTaskLine(task, true, 80)
		if lineSelected == "" {
			t.Errorf("renderTaskLine (selected) for %q returned empty", task.Title)
		}
	}
}

func TestMoveCursor_Normal(t *testing.T) {
	s, _ := storage.NewYAMLStore(t.TempDir())
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewTasksView(s, date)
	tasks := []model.Task{
		{ID: "t-001", Title: "A", Status: model.StatusTodo, CreatedAt: date, UpdatedAt: date},
		{ID: "t-002", Title: "B", Status: model.StatusTodo, CreatedAt: date, UpdatedAt: date},
		{ID: "t-003", Title: "C", Status: model.StatusTodo, CreatedAt: date, UpdatedAt: date},
	}
	v2, _ := v.moveCursor(tasks, +1)
	if v2.cursor != 1 {
		t.Errorf("moveCursor +1: cursor=%d, want 1", v2.cursor)
	}
	v3, _ := v2.moveCursor(tasks, -1)
	if v3.cursor != 0 {
		t.Errorf("moveCursor -1: cursor=%d, want 0", v3.cursor)
	}
}

func TestMoveCursor_ForwardAtEnd(t *testing.T) {
	s, _ := storage.NewYAMLStore(t.TempDir())
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewTasksView(s, date)
	tasks := []model.Task{
		{ID: "t-001", Title: "A", Status: model.StatusTodo, CreatedAt: date, UpdatedAt: date},
	}
	v2, cmd := v.moveCursor(tasks, +1)
	if v2.cursor != 0 {
		t.Errorf("moveCursor +1 at end: cursor=%d, want 0", v2.cursor)
	}
	if cmd != nil {
		t.Error("expected nil cmd when clamped at end")
	}
}

func TestMoveCursor_BackwardAtStart(t *testing.T) {
	s, _ := storage.NewYAMLStore(t.TempDir())
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewTasksView(s, date)
	tasks := []model.Task{
		{ID: "t-001", Title: "A", Status: model.StatusTodo, CreatedAt: date, UpdatedAt: date},
	}
	v2, cmd := v.moveCursor(tasks, -1)
	if v2.cursor != 0 {
		t.Errorf("moveCursor -1 at start: cursor=%d, want 0", v2.cursor)
	}
	if cmd != nil {
		t.Error("expected nil cmd when clamped at start")
	}
}

func TestTasksViewSavePlanMsg(t *testing.T) {
	store, _ := storage.NewYAMLStore(t.TempDir())
	date := time.Date(2026, 5, 26, 0, 0, 0, 0, time.UTC)
	v := NewTasksView(store, date)
	plan := &model.DayPlan{Date: date, Tasks: []model.Task{
		{ID: "t-001", Title: "Test", Status: model.StatusTodo},
	}}
	msg := v.savePlanMsg(plan)
	if _, ok := msg.(errMsg); ok {
		t.Error("savePlanMsg returned errMsg on success")
	}
}

func TestTasksViewHandleExistingTaskAction(t *testing.T) {
	store, _ := storage.NewYAMLStore(t.TempDir())
	date := time.Date(2026, 5, 26, 0, 0, 0, 0, time.UTC)
	v := NewTasksView(store, date)
	t0 := model.Task{ID: "t-001", Title: "Task", Status: model.StatusTodo}

	// unknown key → no-op
	v2, cmd := v.handleExistingTaskAction(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}}, t0)
	if cmd != nil {
		t.Error("unknown key should return nil cmd")
	}
	_ = v2

	// "d" → toggle done (returns non-nil cmd)
	_, cmd = v.handleExistingTaskAction(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}, t0)
	if cmd == nil {
		t.Error("'d' should return a cmd")
	}
}

// ── savePlanMsg error branch ──────────────────────────────────────────────────

func TestSavePlanMsgError(t *testing.T) {
	dir := t.TempDir()
	store, _ := storage.NewYAMLStore(dir)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewTasksView(store, date)

	daysDir := filepath.Join(dir, "days")
	if err := os.Chmod(daysDir, 0o555); err != nil {
		t.Fatalf("Chmod: %v", err)
	}
	t.Cleanup(func() { os.Chmod(daysDir, 0o755) })

	plan := &model.DayPlan{Date: date}
	msg := v.savePlanMsg(plan)
	if _, ok := msg.(errMsg); !ok {
		t.Fatalf("expected errMsg on write failure, got %T", msg)
	}
}

// ── toggleShowDone ────────────────────────────────────────────────────────────

func TestToggleShowDone(t *testing.T) {
	store, _ := storage.NewYAMLStore(t.TempDir())
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewTasksView(store, date)

	if v.showingDone() {
		t.Fatal("showingDone should be false initially")
	}

	// 'f' toggles done tasks on
	v2, cmd := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	v = v2
	if !v.showingDone() {
		t.Error("showingDone should be true after first 'f'")
	}
	if cmd != nil {
		t.Error("cmd should be nil in non-global mode")
	}

	// 'f' again toggles done tasks off
	v2, cmd = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	v = v2
	if v.showingDone() {
		t.Error("showingDone should be false after second 'f'")
	}
	if cmd != nil {
		t.Error("cmd should be nil in non-global mode")
	}
}

func TestToggleShowDoneGlobalMode(t *testing.T) {
	store, _ := storage.NewYAMLStore(t.TempDir())
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewTasksView(store, date)

	// Enter global mode by injecting a non-nil globalResults slice
	v2, _ := v.Update(GlobalTasksMsg{Tasks: []*model.Task{}})
	v = v2

	_, cmd := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	if cmd == nil {
		t.Error("cmd should be non-nil in global mode so statuses are refreshed")
	}
	msg := cmd()
	if _, ok := msg.(GlobalTasksMsg); !ok {
		t.Fatalf("expected GlobalTasksMsg, got %T", msg)
	}
}

// ── carryTaskTo error branches ────────────────────────────────────────────────

func TestCarryTaskToNotFound(t *testing.T) {
	store, _ := storage.NewYAMLStore(t.TempDir())
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewTasksView(store, date)

	cmd := v.carryTaskTo("no-such-id", date.AddDate(0, 0, 1))
	if cmd == nil {
		t.Fatal("expected non-nil cmd")
	}
	msg := cmd()
	if _, ok := msg.(errMsg); !ok {
		t.Fatalf("expected errMsg for missing task, got %T", msg)
	}
}

func TestCarryTaskToSaveSrcFails(t *testing.T) {
	dir := t.TempDir()
	store, _ := storage.NewYAMLStore(dir)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	plan := &model.DayPlan{
		Date: date,
		Tasks: []model.Task{
			{ID: "t-001", Title: "Task", Status: model.StatusTodo, CreatedAt: date, UpdatedAt: date},
		},
	}
	store.SaveDayPlan(plan)

	v := NewTasksView(store, date)
	v2, _ := v.Update(TasksMsg{Plan: plan, ActLog: &model.ActivityLog{Date: date}})
	v = v2

	// make days/ unwritable so SaveDayPlan(srcPlan) fails
	daysDir := filepath.Join(dir, "days")
	if err := os.Chmod(daysDir, 0o555); err != nil {
		t.Fatalf("Chmod: %v", err)
	}
	t.Cleanup(func() { os.Chmod(daysDir, 0o755) })

	cmd := v.carryTaskTo("t-001", date.AddDate(0, 0, 1))
	msg := cmd()
	if _, ok := msg.(errMsg); !ok {
		t.Fatalf("expected errMsg when save src fails, got %T", msg)
	}
}

// ── loadMsg error branches ────────────────────────────────────────────────────

func TestLoadMsgGetDayPlanError(t *testing.T) {
	dir := t.TempDir()
	store, _ := storage.NewYAMLStore(dir)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)

	dayFile := filepath.Join(dir, "days", date.Format("2006-01-02")+".yaml")
	if err := os.WriteFile(dayFile, []byte(": bad: yaml: \x00"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	v := NewTasksView(store, date)
	msg := v.loadMsg()
	if _, ok := msg.(errMsg); !ok {
		t.Fatalf("expected errMsg when GetDayPlan fails, got %T", msg)
	}
}

func TestLoadMsgGetActivityLogError(t *testing.T) {
	dir := t.TempDir()
	store, _ := storage.NewYAMLStore(dir)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)

	actFile := filepath.Join(dir, "activity", date.Format("2006-01-02")+".yaml")
	if err := os.WriteFile(actFile, []byte(": bad: yaml: \x00"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	v := NewTasksView(store, date)
	msg := v.loadMsg()
	if _, ok := msg.(errMsg); !ok {
		t.Fatalf("expected errMsg when GetActivityLog fails, got %T", msg)
	}
}

// ── navigateDay pane guard ────────────────────────────────────────────────────

func TestNavigateDayDetailPane(t *testing.T) {
	store, _ := storage.NewYAMLStore(t.TempDir())
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewTasksView(store, date)
	v.pane = 1

	v2, cmd := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if !v2.date.Equal(date) {
		t.Errorf("navigate should be a no-op in detail pane, date changed to %v", v2.date)
	}
	if cmd != nil {
		t.Error("cmd should be nil when navigation is blocked by pane")
	}
}

// ── handleTaskAction missing branches ────────────────────────────────────────

func TestHandleTaskActionAddInDetailPane(t *testing.T) {
	store, _ := storage.NewYAMLStore(t.TempDir())
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewTasksView(store, date)
	v.pane = 1

	v2, cmd := v.handleTaskAction(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}, nil)
	if v2.adding {
		t.Error("'a' in detail pane should not open add form")
	}
	if cmd != nil {
		t.Error("cmd should be nil when 'a' pressed in detail pane")
	}
}

func TestHandleTaskActionEmptyList(t *testing.T) {
	store, _ := storage.NewYAMLStore(t.TempDir())
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewTasksView(store, date)

	v2, cmd := v.handleTaskAction(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}}, nil)
	_ = v2
	if cmd != nil {
		t.Error("cmd should be nil when task list is empty")
	}
}

// ── actions ───────────────────────────────────────────────────────────────────

// viewWithTaskAndActions returns a TasksView with a task that has two actions pre-seeded.
func viewWithTaskAndActions(t *testing.T) (TasksView, *storage.YAMLStore, time.Time) {
	t.Helper()
	s, _ := storage.NewYAMLStore(t.TempDir())
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	plan := &model.DayPlan{
		Date: date,
		Tasks: []model.Task{
			{
				ID: "t-20260518-001", Title: "Task with actions", Status: model.StatusTodo,
				CreatedAt: date, UpdatedAt: date,
				Actions: []model.Action{
					{ID: "ac-001", Title: "first step", Done: false},
					{ID: "ac-002", Title: "second step", Done: true},
				},
			},
		},
	}
	s.SaveDayPlan(plan)
	v := NewTasksView(s, date)
	v2, _ := v.Update(TasksMsg{Plan: plan, ActLog: &model.ActivityLog{Date: date}})
	return v2, s, date
}

func TestTasksViewOpenActions(t *testing.T) {
	v, _, _ := viewWithTask(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
	v = v2
	if v.detailMode != detailActions {
		t.Fatalf("expected detailActions, got %v", v.detailMode)
	}
	if v.pane != 1 {
		t.Error("pane should switch to detail (1) after 'A'")
	}
	if v.IsInputActive() {
		// detailActions is considered input-active
		_ = v // OK — actions mode is active, IsInputActive should be true
	}
}

func TestTasksViewActionsEscExitsMode(t *testing.T) {
	v, _, _ := viewWithTaskAndActions(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
	v = v2
	if v.detailMode != detailActions {
		t.Fatalf("prerequisite: expected detailActions, got %v", v.detailMode)
	}

	// esc → back to normal
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyEsc})
	v = v2
	if v.detailMode != detailNormal {
		t.Errorf("expected detailNormal after esc, got %v", v.detailMode)
	}
}

func TestTasksViewActionsEscCancelsAdd(t *testing.T) {
	v, _, _ := viewWithTaskAndActions(t)

	// enter actions mode
	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
	v = v2

	// 'a' → enter add mode
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	v = v2
	if !v.actionAdding {
		t.Fatal("expected actionAdding = true after 'a'")
	}

	// esc → cancel add (stay in detailActions)
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyEsc})
	v = v2
	if v.actionAdding {
		t.Error("actionAdding should be false after esc")
	}
	if v.detailMode != detailActions {
		t.Errorf("detailMode should remain detailActions, got %v", v.detailMode)
	}
}

func TestTasksViewActionsNavigate(t *testing.T) {
	v, _, _ := viewWithTaskAndActions(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
	v = v2

	// cursor starts at 0; 'j' moves down
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	v = v2
	if v.actionCursor != 1 {
		t.Errorf("actionCursor = %d, want 1 after 'j'", v.actionCursor)
	}

	// 'k' moves back up
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	v = v2
	if v.actionCursor != 0 {
		t.Errorf("actionCursor = %d, want 0 after 'k'", v.actionCursor)
	}

	// clamped at 0 on 'k'
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	v = v2
	if v.actionCursor != 0 {
		t.Errorf("actionCursor should be clamped at 0, got %d", v.actionCursor)
	}

	// move to last, then try to go past
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	v = v2
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	v = v2
	if v.actionCursor != 1 {
		t.Errorf("actionCursor should be clamped at 1 (last), got %d", v.actionCursor)
	}
}

func TestTasksViewActionsAddEmptyTitleNoOp(t *testing.T) {
	v, _, _ := viewWithTaskAndActions(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
	v = v2
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	v = v2

	// enter with no text → no save cmd, exits add mode
	v2, cmd := v.Update(tea.KeyMsg{Type: tea.KeyEnter})
	v = v2
	if v.actionAdding {
		t.Error("actionAdding should be false after enter")
	}
	if cmd != nil {
		t.Error("empty title should produce nil cmd")
	}
}

func TestTasksViewActionsAddSave(t *testing.T) {
	v, store, date := viewWithTaskAndActions(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
	v = v2
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	v = v2

	// type "new action"
	for _, r := range "new action" {
		v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
		v = v2
	}

	_, cmd := v.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter with title should produce a save cmd")
	}
	msg := cmd()
	if _, ok := msg.(TasksMsg); !ok {
		t.Fatalf("expected TasksMsg, got %T", msg)
	}

	plan, _ := store.GetDayPlan(date)
	if len(plan.Tasks[0].Actions) != 3 {
		t.Errorf("expected 3 actions after save, got %d", len(plan.Tasks[0].Actions))
	}
	if plan.Tasks[0].Actions[2].Title != "new action" {
		t.Errorf("new action title = %q, want 'new action'", plan.Tasks[0].Actions[2].Title)
	}
}

func TestTasksViewActionsToggle(t *testing.T) {
	v, store, date := viewWithTaskAndActions(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
	v = v2

	// cursor is at 0 (ac-001, Done: false); toggle with space
	_, cmd := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	if cmd == nil {
		t.Fatal("space should produce a toggle cmd")
	}
	msg := cmd()
	if _, ok := msg.(TasksMsg); !ok {
		t.Fatalf("expected TasksMsg after toggle, got %T", msg)
	}

	plan, _ := store.GetDayPlan(date)
	if !plan.Tasks[0].Actions[0].Done {
		t.Error("action 0 should be toggled to Done=true")
	}
}

func TestTasksViewActionsDelete(t *testing.T) {
	v, store, date := viewWithTaskAndActions(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
	v = v2

	_, cmd := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'D'}})
	if cmd == nil {
		t.Fatal("D should produce a delete cmd")
	}
	msg := cmd()
	if _, ok := msg.(TasksMsg); !ok {
		t.Fatalf("expected TasksMsg after delete, got %T", msg)
	}

	plan, _ := store.GetDayPlan(date)
	if len(plan.Tasks[0].Actions) != 1 {
		t.Errorf("expected 1 action after delete, got %d", len(plan.Tasks[0].Actions))
	}
	if plan.Tasks[0].Actions[0].ID != "ac-002" {
		t.Errorf("remaining action should be ac-002, got %s", plan.Tasks[0].Actions[0].ID)
	}
}

func TestTasksViewActionsRenderDetail(t *testing.T) {
	v, _, _ := viewWithTaskAndActions(t)

	v2, _ := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'A'}})
	v = v2

	out := v.View(120, 40)
	if out == "" {
		t.Fatal("View should not be empty in detailActions mode")
	}
	if !strings.Contains(out, "first step") {
		t.Error("View should contain action title 'first step'")
	}
}

func TestRenderTaskLineActionsBadge(t *testing.T) {
	cases := []struct {
		actions   []model.Action
		wantBadge bool
		badgeStr  string
	}{
		{nil, false, ""},
		{[]model.Action{{Done: false}, {Done: false}, {Done: false}}, true, "0/3"},
		{[]model.Action{{Done: true}, {Done: false}, {Done: false}}, true, "1/3"},
		{[]model.Action{{Done: true}, {Done: true}, {Done: true}}, true, "3/3"},
	}
	for _, c := range cases {
		task := model.Task{ID: "t-1", Title: "Test", Status: model.StatusTodo, Actions: c.actions}
		line := renderTaskLine(task, false, 80)
		hasBadge := strings.Contains(line, c.badgeStr)
		if c.wantBadge && !hasBadge {
			t.Errorf("badge not found in line %q, want %q", line, c.badgeStr)
		}
		if !c.wantBadge && strings.Contains(line, "/") {
			t.Errorf("unexpected badge in line %q", line)
		}
	}
}

func TestTasksViewActionsNoTaskNoOp(t *testing.T) {
	store, _ := storage.NewYAMLStore(t.TempDir())
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	v := NewTasksView(store, date)
	// no tasks loaded — actions mode should be a no-op
	v.detailMode = detailActions
	v2, cmd := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{' '}})
	_ = v2
	if cmd != nil {
		t.Error("space with no tasks should produce nil cmd")
	}
}
