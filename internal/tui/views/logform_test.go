package views

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/roramirez/pitlist/internal/model"
)

func TestUpdateLogFormField(t *testing.T) {
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'W'}}

	// Field 0: desc (focused by default in newQuickLogForm)
	f0 := newQuickLogForm("t-001")
	f0.focusIdx = 0
	f0r, _ := updateLogFormField(f0, msg)
	if f0r.desc.Value() == "" {
		t.Error("field 0: desc should have received the key")
	}

	// Field 1: tags — must focus before routing
	f1 := newQuickLogForm("t-001")
	f1.focusIdx = 1
	f1.desc.Blur()
	f1.tags.Focus()
	f1r, _ := updateLogFormField(f1, msg)
	if f1r.tags.Value() == "" {
		t.Error("field 1: tags should have received the key")
	}

	// Fields 2 and 3: just verify no panic
	f2 := newQuickLogForm("t-001")
	f2.focusIdx = 2
	_, _ = updateLogFormField(f2, msg)

	f3 := newQuickLogForm("t-001")
	f3.focusIdx = 3
	_, _ = updateLogFormField(f3, msg)
}

func TestApplyTaskFormFocusContextField(t *testing.T) {
	f := newTaskForm("", "title", "", nil, "", "medium")
	f.focusIdx = 1

	result, blink := applyTaskFormFocus(f)
	_ = result
	if blink {
		t.Error("focusIdx=1 (context) should return blink=false")
	}
}

func TestNextActionID(t *testing.T) {
	cases := []struct {
		actions []model.Action
		want    string
	}{
		{nil, "ac-001"},
		{[]model.Action{}, "ac-001"},
		{[]model.Action{{ID: "ac-001"}}, "ac-002"},
		{make([]model.Action, 9), "ac-010"},
	}
	for _, c := range cases {
		got := nextActionID(c.actions)
		if got != c.want {
			t.Errorf("nextActionID(len=%d) = %q, want %q", len(c.actions), got, c.want)
		}
	}
}

func TestDoneCount(t *testing.T) {
	cases := []struct {
		actions []model.Action
		want    int
	}{
		{nil, 0},
		{[]model.Action{{Done: false}, {Done: false}}, 0},
		{[]model.Action{{Done: true}, {Done: false}}, 1},
		{[]model.Action{{Done: true}, {Done: true}}, 2},
	}
	for _, c := range cases {
		got := doneCount(c.actions)
		if got != c.want {
			t.Errorf("doneCount(%v) = %d, want %d", c.actions, got, c.want)
		}
	}
}

func TestActionBadge(t *testing.T) {
	if actionBadge(nil) != "" {
		t.Error("actionBadge(nil) should be empty")
	}
	if actionBadge([]model.Action{}) != "" {
		t.Error("actionBadge([]) should be empty")
	}
	actions := []model.Action{{Done: true}, {Done: false}, {Done: false}}
	badge := actionBadge(actions)
	if !strings.Contains(badge, "1/3") {
		t.Errorf("actionBadge = %q, want to contain '1/3'", badge)
	}
}

func TestRenderActionsShared(t *testing.T) {
	actions := []model.Action{
		{ID: "ac-001", Title: "step one", Done: true},
		{ID: "ac-002", Title: "step two", Done: false},
	}
	input := textinput.New()

	// Normal mode — not adding
	out := renderActionsShared(actions, 0, false, input, 60)
	if out == "" {
		t.Fatal("renderActionsShared returned empty string")
	}
	if !strings.Contains(out, "step one") {
		t.Error("output should contain action title 'step one'")
	}
	if !strings.Contains(out, "step two") {
		t.Error("output should contain action title 'step two'")
	}
	if !strings.Contains(out, "[x]") {
		t.Error("output should contain '[x]' for done action")
	}
	if !strings.Contains(out, "[ ]") {
		t.Error("output should contain '[ ]' for pending action")
	}

	// Add mode — input row should appear
	out2 := renderActionsShared(actions, 0, true, input, 60)
	if !strings.Contains(out2, "+") {
		t.Error("add mode output should include the input row marker '+'")
	}

	// Empty actions
	out3 := renderActionsShared(nil, 0, false, input, 60)
	if !strings.Contains(out3, "No actions yet") {
		t.Error("empty actions should show 'No actions yet' message")
	}
}

func TestParseScheduleDate(t *testing.T) {
	now := time.Now()
	todayUTC := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	tomorrowUTC := todayUTC.AddDate(0, 0, 1)

	cases := []struct {
		input string
		want  time.Time
	}{
		{"", todayUTC},
		{"today", todayUTC},
		{"TODAY", todayUTC},
		{"tomorrow", tomorrowUTC},
		{"TOMORROW", tomorrowUTC},
		{"2027-01-15", time.Date(2027, 1, 15, 0, 0, 0, 0, time.UTC)},
		{"not-a-date", todayUTC},
	}

	for _, c := range cases {
		got := parseScheduleDate(c.input)
		if !got.Equal(c.want) {
			t.Errorf("parseScheduleDate(%q) = %v, want %v", c.input, got, c.want)
		}
	}
}
