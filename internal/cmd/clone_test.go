package cmd

import (
	"testing"
	"time"

	"github.com/roramirez/pitlist/internal/model"
)

func TestCloneCmdToDate(t *testing.T) {
	s := setupTest(t)
	srcDate := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	destDate := time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC)
	done := srcDate
	seedTask(t, s, srcDate, model.Task{
		ID: "t-20260518-001", Title: "Clone me", Status: model.StatusDone,
		Priority: model.PriorityHigh, Labels: []string{"work"},
		DoneAt: &done, CreatedAt: srcDate, UpdatedAt: srcDate,
		Actions: []model.Action{
			{ID: "ac-001", Title: "step one", Done: true},
			{ID: "ac-002", Title: "step two", Done: false},
		},
	})

	cmd := newCloneCmd()
	cmd.SetArgs([]string{"t-20260518-001", "--to", "2026-05-20"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("clone --to: %v", err)
	}

	// Original untouched on source day
	srcPlan, _ := s.GetDayPlan(srcDate)
	if len(srcPlan.Tasks) != 1 || srcPlan.Tasks[0].ID != "t-20260518-001" {
		t.Errorf("original task should remain on src day, got %v", srcPlan.Tasks)
	}
	if srcPlan.Tasks[0].Status != model.StatusDone {
		t.Errorf("original status changed, got %q", srcPlan.Tasks[0].Status)
	}

	// Clone added to dest day as a skeleton
	destPlan, _ := s.GetDayPlan(destDate)
	if len(destPlan.Tasks) != 1 {
		t.Fatalf("expected 1 task on dest day, got %d", len(destPlan.Tasks))
	}
	clone := destPlan.Tasks[0]
	if clone.ID == "t-20260518-001" {
		t.Errorf("clone should have a fresh ID, got %q", clone.ID)
	}
	if clone.Title != "Clone me" || clone.Priority != model.PriorityHigh {
		t.Errorf("clone did not copy core fields: %+v", clone)
	}
	if clone.Status != model.StatusTodo {
		t.Errorf("clone status = %q, want todo", clone.Status)
	}
	if clone.DoneAt != nil {
		t.Error("clone DoneAt should be nil")
	}
	if len(clone.Actions) != 2 {
		t.Fatalf("clone should keep all actions, got %d", len(clone.Actions))
	}
	for _, a := range clone.Actions {
		if a.Done {
			t.Errorf("clone action %q should be reset to not-done", a.ID)
		}
	}
}

func TestCloneCmdDefaultsToNextDay(t *testing.T) {
	s := setupTest(t)
	srcDate := today()
	seedTask(t, s, srcDate, model.Task{
		ID: "t-clone-001", Title: "Next day", Status: model.StatusTodo,
		CreatedAt: srcDate, UpdatedAt: srcDate,
	})

	cmd := newCloneCmd()
	cmd.SetArgs([]string{"t-clone-001"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("clone: %v", err)
	}

	destPlan, _ := s.GetDayPlan(srcDate.AddDate(0, 0, 1))
	if len(destPlan.Tasks) != 1 || destPlan.Tasks[0].Title != "Next day" {
		t.Errorf("clone not on next day, got %v", destPlan.Tasks)
	}
}

func TestCloneCmdNotFound(t *testing.T) {
	setupTest(t)
	cmd := newCloneCmd()
	cmd.SetArgs([]string{"t-nonexistent-999"})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error for non-existent task ID")
	}
}

func TestCloneCmdInvalidDate(t *testing.T) {
	s := setupTest(t)
	date := today()
	seedTask(t, s, date, model.Task{
		ID: "t-clone-002", Title: "Task", Status: model.StatusTodo,
		CreatedAt: date, UpdatedAt: date,
	})
	cmd := newCloneCmd()
	cmd.SetArgs([]string{"t-clone-002", "--to", "bad-date"})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error for invalid date")
	}
}
