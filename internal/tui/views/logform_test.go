package views

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
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
