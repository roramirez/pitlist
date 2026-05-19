package tui

import (
	"fmt"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/roramirez/pitlist/internal/storage"
)

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
