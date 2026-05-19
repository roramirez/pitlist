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
