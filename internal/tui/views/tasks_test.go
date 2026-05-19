package views

import (
	"testing"

	"github.com/roramirez/pitlist/internal/model"
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
