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
