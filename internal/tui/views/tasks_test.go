package views

import (
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
