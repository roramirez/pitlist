package views

import (
	"fmt"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/roramirez/pitlist/internal/model"
	"github.com/roramirez/pitlist/internal/storage"
)

func TestSearchViewTyping(t *testing.T) {
	v := NewSearchView(nil)
	fmt.Printf("inputFocused: %v  IsInputActive: %v\n", v.inputFocused, v.IsInputActive())

	for _, char := range []rune{'a', 'u', 't', 'h'} {
		msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{char}}
		fmt.Printf("sending %q — msg.String()=%q  len(runes)=%d\n", char, msg.String(), len(msg.Runes))
		v2, _ := v.Update(msg)
		v = v2
		fmt.Printf("  query now: %q\n", v.query)
	}

	if v.query != "auth" {
		t.Errorf("expected query='auth', got %q", v.query)
	}
}

func TestSearchViewBackspace(t *testing.T) {
	v := NewSearchView(nil)
	v.query = "hello"

	msg := tea.KeyMsg{Type: tea.KeyBackspace}
	v2, _ := v.Update(msg)
	v = v2
	if v.query != "hell" {
		t.Errorf("expected 'hell', got %q", v.query)
	}
}

func TestSearchViewBackspaceEmpty(t *testing.T) {
	v := NewSearchView(nil)
	// backspace on empty query should be a no-op
	msg := tea.KeyMsg{Type: tea.KeyBackspace}
	v2, _ := v.Update(msg)
	v = v2
	if v.query != "" {
		t.Errorf("expected empty query, got %q", v.query)
	}
}

func TestSearchViewEscWithNoResults(t *testing.T) {
	v := NewSearchView(nil)
	v.query = "something"
	// esc should always exit input mode so the user can navigate away
	msg := tea.KeyMsg{Type: tea.KeyEsc}
	v2, _ := v.Update(msg)
	v = v2
	if v.inputFocused {
		t.Error("expected inputFocused=false after esc even with no results")
	}
}

func TestSearchViewDownSwitchesToNavigate(t *testing.T) {
	v := NewSearchView(nil)
	v.results = []SearchResult{
		{Kind: SearchResultTask},
	}
	msg := tea.KeyMsg{Type: tea.KeyDown}
	v2, _ := v.Update(msg)
	v = v2
	if v.inputFocused {
		t.Error("expected inputFocused=false after down with results")
	}
	if v.cursor != 0 {
		t.Errorf("expected cursor=0, got %d", v.cursor)
	}
}

func TestSearchViewNavigateJK(t *testing.T) {
	v := NewSearchView(nil)
	v.inputFocused = false
	v.results = make([]SearchResult, 3)
	v.cursor = 1

	// j moves down
	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}}
	v2, _ := v.Update(msg)
	v = v2
	if v.cursor != 2 {
		t.Errorf("j: expected cursor=2, got %d", v.cursor)
	}

	// j at end stays
	v2, _ = v.Update(msg)
	v = v2
	if v.cursor != 2 {
		t.Errorf("j at end: expected cursor=2, got %d", v.cursor)
	}

	// k moves up
	msg = tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	v2, _ = v.Update(msg)
	v = v2
	if v.cursor != 1 {
		t.Errorf("k: expected cursor=1, got %d", v.cursor)
	}
}

func TestSearchViewNavigateKAtTopRestoresFocus(t *testing.T) {
	v := NewSearchView(nil)
	v.inputFocused = false
	v.cursor = 0
	v.results = make([]SearchResult, 2)

	msg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}}
	v2, _ := v.Update(msg)
	v = v2
	if !v.inputFocused {
		t.Error("k at top should restore inputFocused")
	}
}

func TestSearchViewEscInNavigateRestoresFocus(t *testing.T) {
	v := NewSearchView(nil)
	v.inputFocused = false
	v.results = make([]SearchResult, 1)

	msg := tea.KeyMsg{Type: tea.KeyEsc}
	v2, _ := v.Update(msg)
	v = v2
	if !v.inputFocused {
		t.Error("esc in navigate mode should restore inputFocused")
	}
}

func TestSearchViewResultsMsg(t *testing.T) {
	v := NewSearchView(nil)
	v.cursor = 5

	v2, _ := v.Update(SearchResultsMsg{Results: []SearchResult{{Kind: SearchResultTask}}})
	v = v2
	if len(v.results) != 1 {
		t.Errorf("expected 1 result, got %d", len(v.results))
	}
	if v.cursor != 0 {
		t.Errorf("cursor should reset to 0, got %d", v.cursor)
	}
}

func TestDateFromID(t *testing.T) {
	cases := []struct {
		id   string
		want string
	}{
		{"t-20260518-001", "2026-05-18"},
		{"a-20260101-003", "2026-01-01"},
	}
	for _, c := range cases {
		got := dateFromID(c.id)
		if got.Format("2006-01-02") != c.want {
			t.Errorf("dateFromID(%q) = %q, want %q", c.id, got.Format("2006-01-02"), c.want)
		}
	}
}

func TestSearchViewQuery(t *testing.T) {
	v := NewSearchView(nil)
	v.query = "hello"
	if v.Query() != "hello" {
		t.Errorf("Query() = %q, want 'hello'", v.Query())
	}
}

func TestSearchViewSearch(t *testing.T) {
	store, _ := storage.NewYAMLStore(t.TempDir())
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	plan := &model.DayPlan{
		Date: date,
		Tasks: []model.Task{
			{ID: "t-20260518-001", Title: "Auth refactor", Labels: []string{"auth"},
				Status: model.StatusTodo, CreatedAt: date, UpdatedAt: date},
		},
	}
	store.SaveDayPlan(plan)

	ts := time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC)
	al := &model.ActivityLog{
		Date: date,
		Entries: []model.ActivityEntry{
			{ID: "a-20260518-001", Timestamp: ts, Description: "Worked on auth", Tags: []string{"auth"}},
		},
	}
	store.SaveActivityLog(al)

	v := NewSearchView(store)

	// Text search finds task by title
	v.query = "auth"
	cmd := v.search()
	if cmd == nil {
		t.Fatal("search() should return a cmd")
	}
	msg := cmd()
	res, ok := msg.(SearchResultsMsg)
	if !ok {
		t.Fatalf("expected SearchResultsMsg, got %T", msg)
	}
	if len(res.Results) == 0 {
		t.Error("expected at least 1 result for 'auth'")
	}

	// Tag search with # prefix
	v.query = "#auth"
	cmd = v.search()
	msg = cmd()
	res2, ok := msg.(SearchResultsMsg)
	if !ok {
		t.Fatalf("expected SearchResultsMsg, got %T", msg)
	}
	if len(res2.Results) == 0 {
		t.Error("expected at least 1 result for #auth tag search")
	}

	// Empty query returns empty results
	v.query = ""
	cmd = v.search()
	msg = cmd()
	empty, ok := msg.(SearchResultsMsg)
	if !ok {
		t.Fatalf("expected SearchResultsMsg, got %T", msg)
	}
	if len(empty.Results) != 0 {
		t.Error("empty query should return 0 results")
	}
}

func TestSearchViewNavigateEnter(t *testing.T) {
	store, _ := storage.NewYAMLStore(t.TempDir())
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	task := &model.Task{ID: "t-20260518-001", Title: "T", Status: model.StatusTodo}

	v := NewSearchView(store)
	v.results = []SearchResult{{Kind: SearchResultTask, Task: task, Date: date}}
	v.inputFocused = false
	v.cursor = 0

	_, cmd := v.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter in navigate mode should return cmd")
	}
	msg := cmd()
	nav, ok := msg.(SearchNavigateTaskMsg)
	if !ok {
		t.Fatalf("expected SearchNavigateTaskMsg, got %T", msg)
	}
	if !nav.Date.Equal(date) {
		t.Errorf("nav.Date = %v, want %v", nav.Date, date)
	}
}

func TestSearchViewNavigateActivity(t *testing.T) {
	store, _ := storage.NewYAMLStore(t.TempDir())
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	ts := date.Add(10 * time.Hour)
	entry := &model.ActivityEntry{ID: "a-20260518-001", Timestamp: ts, Description: "W"}

	v := NewSearchView(store)
	v.results = []SearchResult{{Kind: SearchResultActivity, Activity: entry, Date: date}}
	v.inputFocused = false
	v.cursor = 0

	_, cmd := v.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd == nil {
		t.Fatal("enter should return cmd for activity result")
	}
	msg := cmd()
	_, ok := msg.(SearchNavigateActivityMsg)
	if !ok {
		t.Fatalf("expected SearchNavigateActivityMsg, got %T", msg)
	}
}

func TestSearchViewView(t *testing.T) {
	store, _ := storage.NewYAMLStore(t.TempDir())
	v := NewSearchView(store)

	// Empty state
	out := v.View(80, 20)
	if out == "" {
		t.Error("View returned empty string")
	}

	// With a query but no results
	v.query = "nothing"
	out = v.View(80, 20)
	if out == "" {
		t.Error("View with query returned empty")
	}

	// With results (task + activity)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	ts := date.Add(10 * time.Hour)
	task := &model.Task{ID: "t-001", Title: "Task", Status: model.StatusTodo, Labels: []string{"work"}}
	entry := &model.ActivityEntry{ID: "a-001", Timestamp: ts, Description: "Work", Tags: []string{"t"}, DurationMin: 30, TaskRef: "t-001"}
	v.results = []SearchResult{
		{Kind: SearchResultTask, Task: task, Date: date},
		{Kind: SearchResultActivity, Activity: entry, Date: date},
	}
	v.inputFocused = false
	out = v.View(80, 20)
	if out == "" {
		t.Error("View with results returned empty")
	}
}

func TestDateFromIDInvalid(t *testing.T) {
	// Short ID falls back to now (just check no panic)
	got := dateFromID("x")
	if got.IsZero() {
		t.Error("expected non-zero time for short id")
	}
}

// ── renderResult done/in-progress task ───────────────────────────────────────

func TestSearchViewRenderResultDoneTask(t *testing.T) {
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	ts := date.Add(10 * time.Hour)

	doneTask := &model.Task{ID: "t-001", Title: "Done task", Status: model.StatusDone}
	inProgTask := &model.Task{ID: "t-002", Title: "In progress task", Status: model.StatusInProgress}

	v := NewSearchView(nil)
	v.results = []SearchResult{
		{Kind: SearchResultTask, Task: doneTask, Date: date},
		{Kind: SearchResultTask, Task: inProgTask, Date: date},
		{Kind: SearchResultActivity, Activity: &model.ActivityEntry{
			ID: "a-001", Timestamp: ts, Description: "Work", DurationMin: 30,
		}, Date: date},
	}
	v.inputFocused = false

	out := v.View(80, 20)
	if out == "" {
		t.Error("View with done/in-progress tasks returned empty")
	}
}

// ── navigate with no results / cursor out of range ───────────────────────────

func TestSearchViewNavigateNoResults(t *testing.T) {
	v := NewSearchView(nil)
	v.inputFocused = false
	v.results = []SearchResult{}
	v.cursor = 0

	cmd := v.navigate()
	if cmd != nil {
		t.Error("navigate() with no results should return nil cmd")
	}

	// cursor out of range
	v.results = make([]SearchResult, 2)
	v.cursor = 5
	cmd = v.navigate()
	if cmd != nil {
		t.Error("navigate() with cursor >= len(results) should return nil cmd")
	}
}

// ── ctrl+h backspace ─────────────────────────────────────────────────────────

func TestSearchViewCtrlHBackspace(t *testing.T) {
	v := NewSearchView(nil)
	v.query = "hello"

	msg := tea.KeyMsg{Type: tea.KeyCtrlH}
	v2, _ := v.Update(msg)
	v = v2
	if v.query != "hell" {
		t.Errorf("ctrl+h: expected 'hell', got %q", v.query)
	}
}

// ── Update WindowSizeMsg ──────────────────────────────────────────────────────

func TestSearchViewWindowSizeMsg(t *testing.T) {
	v := NewSearchView(nil)
	v2, _ := v.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	v = v2
	if v.width != 80 || v.height != 24 {
		t.Errorf("width/height not set: %dx%d", v.width, v.height)
	}
}
