package tui

import (
	"fmt"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/roramirez/pitlist/internal/model"
	"github.com/roramirez/pitlist/internal/storage"
	"github.com/roramirez/pitlist/internal/tui/views"
)

func newTestApp(t *testing.T) App {
	t.Helper()
	store, _ := storage.NewYAMLStore(t.TempDir())
	return NewApp(store)
}

func TestAppSearchTyping(t *testing.T) {
	store, _ := storage.NewYAMLStore(t.TempDir())
	app := NewApp(store)

	// Switch to search tab
	switchMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}}
	model, _ := app.Update(switchMsg)
	app = model.(App)
	fmt.Printf("activeTab after '4': %v (want %v)\n", app.activeTab, tabSearch)
	fmt.Printf("searchView.inputFocused: %v\n", app.searchView.IsInputActive())

	// Type "a"
	typeMsg := tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}}
	model, _ = app.Update(typeMsg)
	app = model.(App)
	fmt.Printf("view after 'a':\n%s\n", app.searchView.View(80, 10))

	if app.searchView.Query() != "a" {
		t.Errorf("expected query='a', got %q", app.searchView.Query())
	}
}

func TestAppTabSwitching(t *testing.T) {
	app := newTestApp(t)

	// '2', '3', '4' each switch from the tasks tab (no active input)
	for _, c := range []struct {
		key  rune
		want tab
	}{
		{'2', tabActivity},
		{'3', tabAgenda},
		{'4', tabSearch},
	} {
		app2 := newTestApp(t) // fresh app each time to stay on tasks tab
		m, _ := app2.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{c.key}})
		got := m.(App).activeTab
		if got != c.want {
			t.Errorf("key %q: activeTab=%v, want %v", c.key, got, c.want)
		}
	}

	// From activity tab, '1' switches back to tasks
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	app = m.(App)
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'1'}})
	app = m.(App)
	if app.activeTab != tabTasks {
		t.Errorf("key '1' from activity: activeTab=%v, want tabTasks", app.activeTab)
	}
}

func TestAppWindowSize(t *testing.T) {
	app := newTestApp(t)
	m, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = m.(App)
	if app.width != 120 || app.height != 40 {
		t.Errorf("width/height not set: got %dx%d", app.width, app.height)
	}
}

func TestAppQuitKey(t *testing.T) {
	app := newTestApp(t)
	_, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	if cmd == nil {
		t.Error("expected non-nil cmd for quit")
	}
}

func TestAppFilterMode(t *testing.T) {
	app := newTestApp(t)
	// '/' opens filter mode when on tasks tab
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	app = m.(App)
	if !app.filterMode {
		t.Error("expected filterMode=true after '/'")
	}

	// esc closes filter mode
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app = m.(App)
	if app.filterMode {
		t.Error("expected filterMode=false after esc")
	}
}

func TestStripANSI(t *testing.T) {
	cases := []struct {
		input string
		want  string
	}{
		{"\x1b[31mred\x1b[0m", "red"},
		{"plain", "plain"},
		{"", ""},
		{"\x1b[1mbold\x1b[m and normal", "bold and normal"},
	}
	for _, c := range cases {
		got := stripANSI(c.input)
		if got != c.want {
			t.Errorf("stripANSI(%q) = %q, want %q", c.input, got, c.want)
		}
	}
}

func TestAppMax(t *testing.T) {
	if max(3, 5) != 5 {
		t.Error("max(3,5) should be 5")
	}
	if max(7, 2) != 7 {
		t.Error("max(7,2) should be 7")
	}
	if max(4, 4) != 4 {
		t.Error("max(4,4) should be 4")
	}
}

func TestAppInit(t *testing.T) {
	app := newTestApp(t)
	cmd := app.Init()
	if cmd == nil {
		t.Error("Init should return a non-nil batch cmd")
	}
}

func TestAppView(t *testing.T) {
	app := newTestApp(t)
	// zero size → "Loading…"
	out := app.View()
	if out != "Loading…" {
		t.Errorf("expected Loading…, got %q", out)
	}

	// non-zero size triggers full render
	m, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = m.(App)
	out = app.View()
	if out == "" {
		t.Error("View with size should return non-empty string")
	}
}

func TestAppViewAllTabs(t *testing.T) {
	app := newTestApp(t)
	m, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = m.(App)

	for _, key := range []rune{'1', '2', '3'} {
		m2, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{key}})
		a := m2.(App)
		if out := a.View(); out == "" {
			t.Errorf("tab %c: View returned empty", key)
		}
	}
}

func TestAppViewFilterOverlay(t *testing.T) {
	app := newTestApp(t)
	m, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = m.(App)
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	app = m.(App)
	if !app.filterMode {
		t.Fatal("expected filterMode=true")
	}
	out := app.View()
	if out == "" {
		t.Error("View with filter overlay should be non-empty")
	}
}

func TestPlaceOverlay(t *testing.T) {
	base := "aaaaaa\nbbbbbb\ncccccc"
	overlay := "XX\nYY"
	result := placeOverlay(base, overlay, 1, 0, 6, 3)
	if result == "" {
		t.Error("placeOverlay returned empty")
	}
}

func TestLabelBadge(t *testing.T) {
	out := labelBadge("work")
	if out == "" {
		t.Error("labelBadge returned empty string")
	}
}

func TestTodayDate(t *testing.T) {
	d := todayDate()
	if d.Hour() != 0 || d.Minute() != 0 || d.Second() != 0 {
		t.Errorf("todayDate should have zeroed time component, got %v", d)
	}
}

func TestAppUpdateSearchResultsMsg(t *testing.T) {
	app := newTestApp(t)
	// Route SearchResultsMsg to searchView
	m, _ := app.Update(views.SearchResultsMsg{})
	app = m.(App)
	if app.activeTab != tabTasks {
		t.Error("tab should not change on SearchResultsMsg")
	}
}

func TestAppUpdateAgendaLoadedMsg(t *testing.T) {
	app := newTestApp(t)
	m, _ := app.Update(views.AgendaLoadedMsg{})
	_ = m.(App)
}

func TestAppUpdateSearchNavigateTask(t *testing.T) {
	app := newTestApp(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	m, _ := app.Update(views.SearchNavigateTaskMsg{Date: date})
	app = m.(App)
	if app.activeTab != tabTasks {
		t.Errorf("expected tabTasks after SearchNavigateTaskMsg, got %v", app.activeTab)
	}
}

func TestAppUpdateSearchNavigateActivity(t *testing.T) {
	app := newTestApp(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	m, _ := app.Update(views.SearchNavigateActivityMsg{Date: date})
	app = m.(App)
	if app.activeTab != tabActivity {
		t.Errorf("expected tabActivity after SearchNavigateActivityMsg, got %v", app.activeTab)
	}
}

func TestAppUpdateAgendaNavigateMsg(t *testing.T) {
	app := newTestApp(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	m, _ := app.Update(views.AgendaNavigateMsg{Date: date})
	app = m.(App)
	if app.activeTab != tabTasks {
		t.Errorf("expected tabTasks after AgendaNavigateMsg, got %v", app.activeTab)
	}
}

func TestAppUpdateFilterApplied(t *testing.T) {
	app := newTestApp(t)
	app.filterMode = true

	m, _ := app.Update(views.FilterAppliedMsg{})
	app = m.(App)
	if app.filterMode {
		t.Error("filterMode should be false after FilterAppliedMsg")
	}
}

func TestAppViewSearchNavigateMode(t *testing.T) {
	// renderStatusBar has a branch for search tab with inputFocused=false
	app := newTestApp(t)
	m, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = m.(App)

	// Switch to search tab (inputFocused starts true)
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}})
	app = m.(App)

	// Force inputFocused=false by injecting a result (with a real task) then pressing down
	task := &model.Task{ID: "t-20260518-001", Title: "T", Status: model.StatusTodo}
	app.searchView, _ = app.searchView.Update(views.SearchResultsMsg{
		Results: []views.SearchResult{{Kind: views.SearchResultTask, Task: task}},
	})
	app.searchView, _ = app.searchView.Update(tea.KeyMsg{Type: tea.KeyDown})

	out := app.View()
	if out == "" {
		t.Error("View in search navigate mode returned empty")
	}
}

func TestAppViewActivityAndAgendaTabs(t *testing.T) {
	app := newTestApp(t)
	m, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = m.(App)

	for _, key := range []rune{'2', '3'} {
		m2, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{key}})
		out := m2.(App).View()
		if out == "" {
			t.Errorf("tab %c: View returned empty", key)
		}
	}
}

func TestAppUpdateTasksMsg(t *testing.T) {
	app := newTestApp(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	m, _ := app.Update(views.TasksMsg{
		Plan:   &model.DayPlan{Date: date, Tasks: []model.Task{}},
		ActLog: &model.ActivityLog{Date: date},
	})
	_ = m.(App)
}

func TestAppUpdateActivityMsg(t *testing.T) {
	app := newTestApp(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	m, _ := app.Update(views.ActivityMsg{Log: &model.ActivityLog{Date: date}})
	_ = m.(App)
}

// ── handleKey "/" on non-tasks tab is a no-op ─────────────────────────────────

func TestAppHandleKeySlashOnNonTasksTab(t *testing.T) {
	app := newTestApp(t)
	// Switch to activity tab first
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	app = m.(App)
	// "/" on activity tab should not enable filterMode
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	app = m.(App)
	if app.filterMode {
		t.Error("filterMode should not activate on non-tasks tab")
	}
}

// ── handleKey dispatches to activity/agenda views ────────────────────────────

func TestAppHandleKeyDispatchToActivityView(t *testing.T) {
	app := newTestApp(t)
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'2'}})
	app = m.(App)
	// Send a key that activity view handles (j = move cursor)
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	_ = m.(App) // must not panic
}

func TestAppHandleKeyDispatchToAgendaView(t *testing.T) {
	app := newTestApp(t)
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'3'}})
	app = m.(App)
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	_ = m.(App)
}

// ── View on search tab with inputFocused=true (renderStatusBar branch) ────────

func TestAppViewSearchTabInputFocused(t *testing.T) {
	app := newTestApp(t)
	m, _ := app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	app = m.(App)
	// Switch to search tab; inputFocused defaults to true
	m, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'4'}})
	app = m.(App)
	if !app.searchView.IsInputActive() {
		t.Fatal("expected search input to be active")
	}
	out := app.View()
	if out == "" {
		t.Error("View on search tab with input focused returned empty")
	}
}

// ── filterMode non-key blink passthrough ─────────────────────────────────────

func TestAppUpdateFilterModeBlinkPassthrough(t *testing.T) {
	app := newTestApp(t)
	// Activate filter mode
	m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	app = m.(App)
	if !app.filterMode {
		t.Fatal("expected filterMode=true")
	}
	// Send a non-key message (WindowSizeMsg) while in filter mode
	m, _ = app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	_ = m.(App) // must not panic
}

// ── Update delegates non-key to active tab ────────────────────────────────────

func TestAppUpdateDelegatesWindowSizeToActiveViews(t *testing.T) {
	for _, key := range []rune{'1', '2', '3', '4'} {
		app := newTestApp(t)
		m, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{key}})
		app = m.(App)
		// WindowSizeMsg is a non-key, delegated to current tab view
		m, _ = app.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
		_ = m.(App)
	}
}
