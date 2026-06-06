package cmd

import (
	"bytes"
	"io"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/roramirez/pitlist/internal/model"
)

func TestShowCmdWithActions(t *testing.T) {
	s := setupTest(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)

	task := model.Task{
		ID:        "t-20260518-001",
		Title:     "Task with checklist",
		Status:    model.StatusTodo,
		Priority:  model.PriorityMedium,
		CreatedAt: date,
		UpdatedAt: date,
		Actions: []model.Action{
			{ID: "ac-001", Title: "first step", Done: false},
			{ID: "ac-002", Title: "second step", Done: true},
		},
	}
	seedTask(t, s, date, task)

	// Capture stdout
	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := newShowCmd()
	cmd.SetArgs([]string{"t-20260518-001"})
	if err := cmd.Execute(); err != nil {
		w.Close()
		os.Stdout = old
		t.Fatalf("show: %v", err)
	}

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	out := buf.String()

	if !strings.Contains(out, "Actions:") {
		t.Error("output should contain 'Actions:' header")
	}
	if !strings.Contains(out, "[ ] first step") {
		t.Errorf("output should contain '[ ] first step', got:\n%s", out)
	}
	if !strings.Contains(out, "[x] second step") {
		t.Errorf("output should contain '[x] second step', got:\n%s", out)
	}
}

func TestShowCmdNoActions(t *testing.T) {
	s := setupTest(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)

	task := model.Task{
		ID:        "t-20260518-001",
		Title:     "Plain task",
		Status:    model.StatusTodo,
		Priority:  model.PriorityMedium,
		CreatedAt: date,
		UpdatedAt: date,
	}
	seedTask(t, s, date, task)

	old := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := newShowCmd()
	cmd.SetArgs([]string{"t-20260518-001"})
	if err := cmd.Execute(); err != nil {
		w.Close()
		os.Stdout = old
		t.Fatalf("show: %v", err)
	}

	w.Close()
	os.Stdout = old

	var buf bytes.Buffer
	io.Copy(&buf, r)
	out := buf.String()

	if strings.Contains(out, "Actions:") {
		t.Error("output should not contain 'Actions:' when task has no actions")
	}
}

func TestShowCmdActionTaskNotFound(t *testing.T) {
	setupTest(t)

	cmd := newShowCmd()
	cmd.SetArgs([]string{"nonexistent-id"})
	if err := cmd.Execute(); err == nil {
		t.Error("show with unknown id should return error")
	}
}
