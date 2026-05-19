package views

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
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
	// esc with no results: stay in input mode
	msg := tea.KeyMsg{Type: tea.KeyEsc}
	v2, _ := v.Update(msg)
	v = v2
	if !v.inputFocused {
		t.Error("expected inputFocused to remain true when no results")
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

func TestDateFromIDInvalid(t *testing.T) {
	// Short ID falls back to now (just check no panic)
	got := dateFromID("x")
	if got.IsZero() {
		t.Error("expected non-zero time for short id")
	}
}
