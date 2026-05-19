package views

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/roramirez/pitlist/internal/model"
	"github.com/roramirez/pitlist/internal/storage"
)

func newTestAgendaStore(t *testing.T) *storage.YAMLStore {
	t.Helper()
	s, err := storage.NewYAMLStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewYAMLStore: %v", err)
	}
	return s
}

func TestNewAgendaView(t *testing.T) {
	store := newTestAgendaStore(t)
	v := NewAgendaView(store)
	if v.store == nil {
		t.Error("store should be set")
	}
	if v.cursor != 0 {
		t.Errorf("cursor should be 0, got %d", v.cursor)
	}
}

func TestAgendaViewLoadedMsg(t *testing.T) {
	store := newTestAgendaStore(t)
	v := NewAgendaView(store)

	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	task := model.Task{ID: "t-20260518-001", Title: "Task A", Status: model.StatusTodo}
	items := []agendaItem{{task: &task, date: date}}

	v2, _ := v.Update(AgendaLoadedMsg{items: items})
	v = v2
	if len(v.items) != 1 {
		t.Errorf("expected 1 item, got %d", len(v.items))
	}
}

func TestAgendaViewCursorClampOnLoad(t *testing.T) {
	store := newTestAgendaStore(t)
	v := NewAgendaView(store)
	v.cursor = 10 // set beyond list size

	v2, _ := v.Update(AgendaLoadedMsg{items: []agendaItem{}})
	v = v2
	if v.cursor != 0 {
		t.Errorf("cursor should clamp to 0, got %d", v.cursor)
	}
}

func TestAgendaViewNavigateJK(t *testing.T) {
	store := newTestAgendaStore(t)
	v := NewAgendaView(store)

	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	task1 := model.Task{ID: "t-20260518-001", Title: "Task 1", Status: model.StatusTodo}
	task2 := model.Task{ID: "t-20260518-002", Title: "Task 2", Status: model.StatusTodo}
	v2, _ := v.Update(AgendaLoadedMsg{items: []agendaItem{
		{task: &task1, date: date},
		{task: &task2, date: date},
	}})
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
		t.Errorf("j at end: expected cursor=1, got %d", v.cursor)
	}

	// k moves up
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	v = v2
	if v.cursor != 0 {
		t.Errorf("k: expected cursor=0, got %d", v.cursor)
	}

	// k at top stays
	v2, _ = v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	v = v2
	if v.cursor != 0 {
		t.Errorf("k at top: expected cursor=0, got %d", v.cursor)
	}
}

func TestAgendaViewNavigateEnter(t *testing.T) {
	store := newTestAgendaStore(t)
	v := NewAgendaView(store)

	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	task := model.Task{ID: "t-20260518-001", Title: "Task", Status: model.StatusTodo}
	v2, _ := v.Update(AgendaLoadedMsg{items: []agendaItem{{task: &task, date: date}}})
	v = v2

	_, cmd := v.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("expected non-nil cmd for enter")
	}
	msg := cmd()
	nav, ok := msg.(AgendaNavigateMsg)
	if !ok {
		t.Fatalf("expected AgendaNavigateMsg, got %T", msg)
	}
	if !nav.Date.Equal(date) {
		t.Errorf("navigate date = %v, want %v", nav.Date, date)
	}
}

func TestAgendaViewRefresh(t *testing.T) {
	store := newTestAgendaStore(t)
	v := NewAgendaView(store)

	_, cmd := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if cmd == nil {
		t.Error("r should return a load command")
	}
}

func TestAgendaViewWindowSize(t *testing.T) {
	store := newTestAgendaStore(t)
	v := NewAgendaView(store)

	v2, _ := v.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	v = v2
	if v.width != 100 || v.height != 30 {
		t.Errorf("width/height not set: %dx%d", v.width, v.height)
	}
}

func TestRenderAgendaTask(t *testing.T) {
	task := &model.Task{
		ID:     "t-20260518-001",
		Title:  "My task",
		Status: model.StatusTodo,
	}
	line := renderAgendaTask(task, false)
	if line == "" {
		t.Error("expected non-empty rendered line")
	}

	// InProgress shows [~]
	task.Status = model.StatusInProgress
	line = renderAgendaTask(task, false)
	if line == "" {
		t.Error("expected non-empty line for in_progress task")
	}

	// High priority includes "!"
	task.Priority = model.PriorityHigh
	task.Status = model.StatusTodo
	line = renderAgendaTask(task, false)
	if line == "" {
		t.Error("expected non-empty line for high priority task")
	}

	// Carried task
	task.CarryFrom = "2026-05-17"
	line = renderAgendaTask(task, false)
	if line == "" {
		t.Error("expected non-empty line for carried task")
	}

	// Selected applies styling
	line = renderAgendaTask(task, true)
	if line == "" {
		t.Error("expected non-empty line for selected task")
	}
}

func TestAgendaViewLoadItems(t *testing.T) {
	store := newTestAgendaStore(t)
	// Seed a task for today and one done task (should be excluded)
	today := agendaToday()
	plan := &model.DayPlan{
		Date: today,
		Tasks: []model.Task{
			{ID: "t-pending", Title: "Pending", Status: model.StatusTodo, CreatedAt: today, UpdatedAt: today},
			{ID: "t-done", Title: "Done", Status: model.StatusDone, CreatedAt: today, UpdatedAt: today},
		},
	}
	store.SaveDayPlan(plan)

	v := NewAgendaView(store)
	cmd := v.Load()
	if cmd == nil {
		t.Fatal("Load should return a cmd")
	}
	msg := cmd()
	loaded, ok := msg.(AgendaLoadedMsg)
	if !ok {
		t.Fatalf("expected AgendaLoadedMsg, got %T", msg)
	}
	// Only the pending task should appear
	found := false
	for _, item := range loaded.items {
		if item.task.ID == "t-pending" {
			found = true
		}
		if item.task.ID == "t-done" {
			t.Error("done task should not appear in agenda")
		}
	}
	if !found {
		t.Error("pending task not found in agenda")
	}
}

func TestAgendaViewMarkDone(t *testing.T) {
	store := newTestAgendaStore(t)
	date := agendaToday()
	plan := &model.DayPlan{
		Date: date,
		Tasks: []model.Task{
			{ID: "t-mark", Title: "Mark done", Status: model.StatusTodo, CreatedAt: date, UpdatedAt: date},
		},
	}
	store.SaveDayPlan(plan)

	task := &model.Task{ID: "t-mark", Title: "Mark done", Status: model.StatusTodo}
	item := agendaItem{task: task, date: date}

	v := NewAgendaView(store)
	cmd := v.markDone(item)
	if cmd == nil {
		t.Fatal("markDone should return a cmd")
	}
	msg := cmd()
	if _, ok := msg.(AgendaLoadedMsg); !ok {
		t.Fatalf("expected AgendaLoadedMsg after markDone, got %T", msg)
	}

	// Verify task is done in store
	updated, _ := store.GetDayPlan(date)
	if updated.Tasks[0].Status != model.StatusDone {
		t.Errorf("task status = %q, want done", updated.Tasks[0].Status)
	}
}

func TestAgendaViewRenderDayHeader(t *testing.T) {
	store := newTestAgendaStore(t)
	v := NewAgendaView(store)
	today := agendaToday()

	// today
	out := v.renderDayHeader(today, today)
	if out == "" {
		t.Error("renderDayHeader(today) returned empty")
	}
	// yesterday (overdue)
	out = v.renderDayHeader(today.AddDate(0, 0, -1), today)
	if out == "" {
		t.Error("renderDayHeader(overdue) returned empty")
	}
	// tomorrow
	out = v.renderDayHeader(today.AddDate(0, 0, 1), today)
	if out == "" {
		t.Error("renderDayHeader(tomorrow) returned empty")
	}
	// future
	out = v.renderDayHeader(today.AddDate(0, 0, 3), today)
	if out == "" {
		t.Error("renderDayHeader(future) returned empty")
	}
}

func TestAgendaViewView(t *testing.T) {
	store := newTestAgendaStore(t)
	v := NewAgendaView(store)

	// Empty agenda
	out := v.View(80, 20)
	if out == "" {
		t.Error("View (empty) returned empty")
	}

	// With items
	date := agendaToday()
	task := model.Task{ID: "t-001", Title: "Task", Status: model.StatusTodo,
		Priority: model.PriorityHigh, CarryFrom: "prev", Labels: []string{"work"}}
	v2, _ := v.Update(AgendaLoadedMsg{items: []agendaItem{{task: &task, date: date}}})
	v = v2
	out = v.View(80, 20)
	if out == "" {
		t.Error("View (with items) returned empty")
	}
}

func TestAgendaViewHandleKeyDWithoutItems(t *testing.T) {
	store := newTestAgendaStore(t)
	v := NewAgendaView(store)
	// 'd' with no items should be a no-op
	_, cmd := v.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	if cmd != nil {
		// cmd is allowed (Load), just shouldn't panic
	}
}

func TestAgendaAdjustScroll(t *testing.T) {
	v := AgendaView{height: 10}
	v.cursor = 5
	v.adjustScroll()
	// cursor=5, visibleLines=6 (height-4), so scroll should be 0
	if v.scroll != 0 {
		t.Errorf("unexpected scroll: %d", v.scroll)
	}

	v.cursor = 20
	v.adjustScroll()
	if v.scroll < 1 {
		t.Errorf("scroll should advance when cursor is beyond window, got %d", v.scroll)
	}
}
