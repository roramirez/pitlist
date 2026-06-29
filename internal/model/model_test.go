package model

import (
	"testing"
	"time"
)

func TestStatusConstants(t *testing.T) {
	if StatusTodo != "todo" {
		t.Errorf("StatusTodo = %q", StatusTodo)
	}
	if StatusInProgress != "in_progress" {
		t.Errorf("StatusInProgress = %q", StatusInProgress)
	}
	if StatusDone != "done" {
		t.Errorf("StatusDone = %q", StatusDone)
	}
	if StatusCancelled != "cancelled" {
		t.Errorf("StatusCancelled = %q", StatusCancelled)
	}
}

func TestPriorityConstants(t *testing.T) {
	if PriorityLow != "low" {
		t.Errorf("PriorityLow = %q", PriorityLow)
	}
	if PriorityMedium != "medium" {
		t.Errorf("PriorityMedium = %q", PriorityMedium)
	}
	if PriorityHigh != "high" {
		t.Errorf("PriorityHigh = %q", PriorityHigh)
	}
}

func TestTaskZeroValue(t *testing.T) {
	var task Task
	if task.Status != "" {
		t.Errorf("zero Task.Status should be empty string, got %q", task.Status)
	}
	if task.ActivityRefs != nil {
		t.Error("zero Task.ActivityRefs should be nil")
	}
}

func TestTaskWithFields(t *testing.T) {
	now := time.Now().UTC()
	done := now
	task := Task{
		ID:        "t-20260518-001",
		Title:     "Test",
		Context:   "work",
		Notes:     "some notes",
		Labels:    []string{"a", "b"},
		Status:    StatusInProgress,
		Priority:  PriorityHigh,
		CreatedAt: now,
		UpdatedAt: now,
		DoneAt:    &done,
		DueDate:   "2026-12-31",
		CarryFrom: "2026-05-17",
		CarryTo:   "2026-05-19",
		ActivityRefs: []ActivityRef{
			{ID: "a-001", Date: "2026-05-18"},
		},
	}

	if task.ID != "t-20260518-001" {
		t.Errorf("ID = %q", task.ID)
	}
	if task.Status != StatusInProgress {
		t.Errorf("Status = %q", task.Status)
	}
	if len(task.Labels) != 2 {
		t.Errorf("Labels len = %d", len(task.Labels))
	}
	if task.DoneAt == nil || !task.DoneAt.Equal(done) {
		t.Error("DoneAt mismatch")
	}
	if len(task.ActivityRefs) != 1 || task.ActivityRefs[0].ID != "a-001" {
		t.Errorf("ActivityRefs = %v", task.ActivityRefs)
	}
}

func TestDayPlan(t *testing.T) {
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	plan := DayPlan{
		Date:  date,
		Tasks: []Task{{ID: "t-001", Title: "T1"}},
	}
	if !plan.Date.Equal(date) {
		t.Errorf("Date = %v", plan.Date)
	}
	if len(plan.Tasks) != 1 {
		t.Errorf("Tasks len = %d", len(plan.Tasks))
	}
}

func TestActivityEntry(t *testing.T) {
	ts := time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC)
	e := ActivityEntry{
		ID:          "a-20260518-001",
		Timestamp:   ts,
		Description: "Did work",
		Tags:        []string{"work", "debug"},
		TaskRef:     "t-20260518-001",
		DurationMin: 45,
	}
	if e.DurationMin != 45 {
		t.Errorf("DurationMin = %d", e.DurationMin)
	}
	if len(e.Tags) != 2 {
		t.Errorf("Tags len = %d", len(e.Tags))
	}
}

func TestActivityLog(t *testing.T) {
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	log := ActivityLog{
		Date:    date,
		Entries: []ActivityEntry{{ID: "a-001"}},
	}
	if len(log.Entries) != 1 {
		t.Errorf("Entries len = %d", len(log.Entries))
	}
}

func TestActivityRef(t *testing.T) {
	ref := ActivityRef{ID: "a-20260518-001", Date: "2026-05-18"}
	if ref.ID != "a-20260518-001" {
		t.Errorf("ID = %q", ref.ID)
	}
	if ref.Date != "2026-05-18" {
		t.Errorf("Date = %q", ref.Date)
	}
}

func TestCloneSkeleton(t *testing.T) {
	now := time.Now().UTC()
	src := Task{
		ID:        "t-20260518-001",
		Title:     "Original",
		Context:   "work",
		Notes:     "keep notes",
		Labels:    []string{"a", "b"},
		Status:    StatusDone,
		Priority:  PriorityHigh,
		CreatedAt: now,
		UpdatedAt: now,
		DoneAt:    &now,
		CarryFrom: "2026-05-17",
		CarryTo:   "2026-05-19",
		ActivityRefs: []ActivityRef{
			{ID: "a-001", Date: "2026-05-18"},
		},
		Actions: []Action{
			{ID: "ac-001", Title: "step one", Done: true},
			{ID: "ac-002", Title: "step two", Done: false},
		},
	}

	clone := src.CloneSkeleton()

	if clone.ID != "" {
		t.Errorf("ID = %q, want empty", clone.ID)
	}
	if clone.Status != StatusTodo {
		t.Errorf("Status = %q, want todo", clone.Status)
	}
	if clone.DoneAt != nil {
		t.Error("DoneAt should be nil")
	}
	if clone.CarryFrom != "" || clone.CarryTo != "" {
		t.Errorf("carry metadata not cleared: %q %q", clone.CarryFrom, clone.CarryTo)
	}
	if clone.ActivityRefs != nil {
		t.Error("ActivityRefs should be nil")
	}
	if !clone.CreatedAt.IsZero() || !clone.UpdatedAt.IsZero() {
		t.Error("timestamps should be zeroed")
	}
	if clone.Title != "Original" || clone.Notes != "keep notes" || clone.Priority != PriorityHigh {
		t.Errorf("core fields not preserved: %+v", clone)
	}
	if len(clone.Actions) != 2 {
		t.Fatalf("expected 2 actions, got %d", len(clone.Actions))
	}
	for _, a := range clone.Actions {
		if a.Done {
			t.Errorf("action %q should be reset to not-done", a.ID)
		}
	}

	// Deep-copy: mutating clone must not affect the source.
	clone.Actions[0].Done = true
	clone.Labels[0] = "mutated"
	if src.Actions[0].Done != true {
		// src action 0 was originally Done=true; ensure it's still true (unchanged by clone reset)
		t.Error("source action should be unchanged")
	}
	if src.Labels[0] != "a" {
		t.Errorf("source labels mutated via clone: %v", src.Labels)
	}
}

func TestCloneSkeletonNoActions(t *testing.T) {
	src := Task{ID: "t-1", Title: "No actions", Status: StatusTodo}
	clone := src.CloneSkeleton()
	if clone.Actions != nil {
		t.Errorf("Actions should be nil when source has none, got %v", clone.Actions)
	}
}
