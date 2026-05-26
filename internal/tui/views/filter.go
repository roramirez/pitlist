package views

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/roramirez/pitlist/internal/model"
)

type FilterAppliedMsg struct {
	Filter TaskFilter
}

type FilterView struct {
	active       bool
	search       textinput.Model
	labelInput   textinput.Model
	focusIdx     int
	showDone     bool
	showTodo     bool
	showProgress bool
}

func (v *FilterView) Activate() tea.Cmd {
	v.active = true
	v.focusIdx = 0
	v.search.Focus()
	return textinput.Blink
}
func (v FilterView) IsActive() bool { return v.active }

func NewFilterView() FilterView {
	search := textinput.New()
	search.Placeholder = "Search tasks…"
	search.Focus()

	li := textinput.New()
	li.Placeholder = "Labels (space-separated)"

	return FilterView{
		search:       search,
		labelInput:   li,
		showTodo:     true,
		showProgress: true,
		showDone:     false,
	}
}

func (v FilterView) Update(msg tea.Msg) (FilterView, tea.Cmd) {
	if !v.active {
		return v, nil
	}
	switch msg := msg.(type) {
	case tea.KeyMsg:
		return v.handleKeyMsg(msg)
	default:
		return v.updateActiveInput(msg)
	}
}

func (v FilterView) handleKeyMsg(msg tea.KeyMsg) (FilterView, tea.Cmd) {
	switch msg.String() {
	case "esc":
		v.active = false
	case "enter":
		v.active = false
		return v, v.apply()
	case "tab":
		v.focusIdx = (v.focusIdx + 1) % 3
		return v.focus()
	case "1", "2", "3":
		v = v.toggleStatusFilter(msg.String())
	default:
		return v.updateActiveInput(msg)
	}
	return v, nil
}

func (v FilterView) toggleStatusFilter(key string) FilterView {
	if v.focusIdx != 2 {
		return v
	}
	switch key {
	case "1":
		v.showTodo = !v.showTodo
	case "2":
		v.showProgress = !v.showProgress
	case "3":
		v.showDone = !v.showDone
	}
	return v
}

func (v FilterView) updateActiveInput(msg tea.Msg) (FilterView, tea.Cmd) {
	var cmd tea.Cmd
	switch v.focusIdx {
	case 0:
		v.search, cmd = v.search.Update(msg)
	case 1:
		v.labelInput, cmd = v.labelInput.Update(msg)
	}
	return v, cmd
}

func (v FilterView) focus() (FilterView, tea.Cmd) {
	v.search.Blur()
	v.labelInput.Blur()
	switch v.focusIdx {
	case 0:
		v.search.Focus()
	case 1:
		v.labelInput.Focus()
	}
	return v, textinput.Blink
}

func (v FilterView) apply() tea.Cmd {
	return func() tea.Msg {
		var statuses []model.TaskStatus
		if v.showTodo {
			statuses = append(statuses, model.StatusTodo)
		}
		if v.showProgress {
			statuses = append(statuses, model.StatusInProgress)
		}
		if v.showDone {
			statuses = append(statuses, model.StatusDone)
		}
		if len(statuses) == 0 {
			statuses = []model.TaskStatus{model.StatusTodo, model.StatusInProgress}
		}

		var labels []string
		for _, l := range strings.Fields(v.labelInput.Value()) {
			labels = append(labels, l)
		}

		return FilterAppliedMsg{Filter: TaskFilter{
			Search:   v.search.Value(),
			Labels:   labels,
			Statuses: statuses,
		}}
	}
}

func (v FilterView) View() string {
	fieldLabel := func(idx int, label string) string {
		if v.focusIdx == idx {
			return sTitle.Render("> " + label)
		}
		return sMuted.Render("  " + label)
	}

	check := func(on bool) string {
		if on {
			return "[x]"
		}
		return "[ ]"
	}

	statusLine := fmt.Sprintf("  Status: %s todo  %s in_progress  %s done  (1/2/3 to toggle)",
		check(v.showTodo), check(v.showProgress), check(v.showDone),
	)

	content := strings.Join([]string{
		sTitle.Render("─── Filter ───"),
		fieldLabel(0, "Search:") + "  " + v.search.View(),
		fieldLabel(1, "Labels:") + "  " + v.labelInput.View(),
		fieldLabel(2, "") + statusLine,
		"",
		sMuted.Render("  enter to apply  esc to cancel  tab next field"),
	}, "\n")

	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Padding(1, 2).
		Render(content)
}
