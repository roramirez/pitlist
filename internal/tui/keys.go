package tui

import "github.com/charmbracelet/bubbles/key"

type keyMap struct {
	Up       key.Binding
	Down     key.Binding
	Left     key.Binding
	Right    key.Binding
	Tab      key.Binding
	Add      key.Binding
	Done     key.Binding
	Edit     key.Binding
	Carry    key.Binding
	Delete   key.Binding
	Filter   key.Binding
	Week     key.Binding
	Log      key.Binding
	Tasks    key.Binding
	Confirm  key.Binding
	Cancel   key.Binding
	Quit     key.Binding
	Help     key.Binding
}

var keys = keyMap{
	Up:      key.NewBinding(key.WithKeys("k", "up"), key.WithHelp("k/↑", "up")),
	Down:    key.NewBinding(key.WithKeys("j", "down"), key.WithHelp("j/↓", "down")),
	Left:    key.NewBinding(key.WithKeys("h", "left", "["), key.WithHelp("h/←", "prev day")),
	Right:   key.NewBinding(key.WithKeys("l", "right", "]"), key.WithHelp("l/→", "next day")),
	Tab:     key.NewBinding(key.WithKeys("tab"), key.WithHelp("tab", "switch pane")),
	Add:     key.NewBinding(key.WithKeys("a"), key.WithHelp("a", "add")),
	Done:    key.NewBinding(key.WithKeys("d"), key.WithHelp("d", "done")),
	Edit:    key.NewBinding(key.WithKeys("e"), key.WithHelp("e", "edit")),
	Carry:   key.NewBinding(key.WithKeys("c"), key.WithHelp("c", "carry")),
	Delete:  key.NewBinding(key.WithKeys("D"), key.WithHelp("D", "delete")),
	Filter:  key.NewBinding(key.WithKeys("/"), key.WithHelp("/", "filter")),
	Week:    key.NewBinding(key.WithKeys("w"), key.WithHelp("w", "week view")),
	Log:     key.NewBinding(key.WithKeys("2"), key.WithHelp("2", "activity log")),
	Tasks:   key.NewBinding(key.WithKeys("1"), key.WithHelp("1", "tasks")),
	Confirm: key.NewBinding(key.WithKeys("enter", "ctrl+s"), key.WithHelp("enter", "confirm")),
	Cancel:  key.NewBinding(key.WithKeys("esc"), key.WithHelp("esc", "cancel")),
	Quit:    key.NewBinding(key.WithKeys("q", "ctrl+c"), key.WithHelp("q", "quit")),
	Help:    key.NewBinding(key.WithKeys("?"), key.WithHelp("?", "help")),
}
