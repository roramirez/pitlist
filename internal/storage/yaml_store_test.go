package storage

import (
	"testing"
	"time"

	"github.com/roramirez/pitlist/internal/model"
)

func newTestStore(t *testing.T) *YAMLStore {
	t.Helper()
	s, err := NewYAMLStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewYAMLStore: %v", err)
	}
	return s
}

func TestDayPlanRoundTrip(t *testing.T) {
	s := newTestStore(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)

	plan := &model.DayPlan{
		Date: date,
		Tasks: []model.Task{
			{
				ID:        "t-20260518-001",
				Title:     "Test task",
				Labels:    []string{"work"},
				Status:    model.StatusTodo,
				Priority:  model.PriorityHigh,
				CreatedAt: date,
				UpdatedAt: date,
			},
		},
	}

	if err := s.SaveDayPlan(plan); err != nil {
		t.Fatalf("SaveDayPlan: %v", err)
	}

	got, err := s.GetDayPlan(date)
	if err != nil {
		t.Fatalf("GetDayPlan: %v", err)
	}

	if len(got.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(got.Tasks))
	}
	if got.Tasks[0].Title != "Test task" {
		t.Errorf("title mismatch: got %q", got.Tasks[0].Title)
	}
	if got.Tasks[0].Status != model.StatusTodo {
		t.Errorf("status mismatch: got %q", got.Tasks[0].Status)
	}
}

func TestGetDayPlanMissing(t *testing.T) {
	s := newTestStore(t)
	date := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	plan, err := s.GetDayPlan(date)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plan.Tasks) != 0 {
		t.Errorf("expected empty plan, got %d tasks", len(plan.Tasks))
	}
}

func TestGetTaskByID(t *testing.T) {
	s := newTestStore(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	plan := &model.DayPlan{
		Date: date,
		Tasks: []model.Task{
			{ID: "t-20260518-001", Title: "Find me", Status: model.StatusTodo, CreatedAt: date, UpdatedAt: date},
		},
	}
	if err := s.SaveDayPlan(plan); err != nil {
		t.Fatal(err)
	}

	task, _, err := s.GetTaskByID("t-20260518-001")
	if err != nil {
		t.Fatalf("GetTaskByID: %v", err)
	}
	if task.Title != "Find me" {
		t.Errorf("unexpected title: %q", task.Title)
	}
}

func TestListTasksFilterByLabel(t *testing.T) {
	s := newTestStore(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	plan := &model.DayPlan{
		Date: date,
		Tasks: []model.Task{
			{ID: "t-20260518-001", Title: "Work task", Labels: []string{"work"}, Status: model.StatusTodo, CreatedAt: date, UpdatedAt: date},
			{ID: "t-20260518-002", Title: "Personal task", Labels: []string{"personal"}, Status: model.StatusTodo, CreatedAt: date, UpdatedAt: date},
		},
	}
	if err := s.SaveDayPlan(plan); err != nil {
		t.Fatal(err)
	}

	results, err := s.ListTasks(TaskFilter{Labels: []string{"work"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1, got %d", len(results))
	}
	if results[0].Title != "Work task" {
		t.Errorf("unexpected task: %q", results[0].Title)
	}
}

func TestListTasksFilterByStatus(t *testing.T) {
	s := newTestStore(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	plan := &model.DayPlan{
		Date: date,
		Tasks: []model.Task{
			{ID: "t-20260518-001", Title: "Todo", Status: model.StatusTodo, CreatedAt: date, UpdatedAt: date},
			{ID: "t-20260518-002", Title: "Done", Status: model.StatusDone, CreatedAt: date, UpdatedAt: date},
		},
	}
	if err := s.SaveDayPlan(plan); err != nil {
		t.Fatal(err)
	}

	results, err := s.ListTasks(TaskFilter{Statuses: []model.TaskStatus{model.StatusDone}})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Title != "Done" {
		t.Errorf("unexpected results: %v", results)
	}
}

func TestActivityLogRoundTrip(t *testing.T) {
	s := newTestStore(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	ts := time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC)

	log := &model.ActivityLog{
		Date: date,
		Entries: []model.ActivityEntry{
			{
				ID:          "a-20260518-001",
				Timestamp:   ts,
				Description: "Did something useful",
				Tags:        []string{"work"},
				DurationMin: 30,
			},
		},
	}

	if err := s.SaveActivityLog(log); err != nil {
		t.Fatal(err)
	}

	got, err := s.GetActivityLog(date)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(got.Entries))
	}
	if got.Entries[0].Description != "Did something useful" {
		t.Errorf("description mismatch: %q", got.Entries[0].Description)
	}
}

func TestListActivityFilterByTag(t *testing.T) {
	s := newTestStore(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	ts := time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC)

	log := &model.ActivityLog{
		Date: date,
		Entries: []model.ActivityEntry{
			{ID: "a-20260518-001", Timestamp: ts, Description: "Debugging", Tags: []string{"debugging"}},
			{ID: "a-20260518-002", Timestamp: ts, Description: "Meeting", Tags: []string{"meetings"}},
		},
	}
	if err := s.SaveActivityLog(log); err != nil {
		t.Fatal(err)
	}

	results, err := s.ListActivity(ActivityFilter{Tags: []string{"debugging"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Description != "Debugging" {
		t.Errorf("unexpected results: %v", results)
	}
}

func TestNextIDs(t *testing.T) {
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	plan := &model.DayPlan{Date: date, Tasks: []model.Task{}}
	id := NextTaskID(plan)
	if id != "t-20260518-001" {
		t.Errorf("unexpected task ID: %q", id)
	}

	log := &model.ActivityLog{Date: date, Entries: []model.ActivityEntry{}}
	aid := NextActivityID(log)
	if aid != "a-20260518-001" {
		t.Errorf("unexpected activity ID: %q", aid)
	}
}
