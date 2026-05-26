package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/roramirez/pitlist/internal/storage"
	"github.com/roramirez/pitlist/internal/tui/views"
)

type tab int

const (
	tabTasks tab = iota
	tabActivity
	tabAgenda
	tabSearch
	tabFuture
)

type App struct {
	store        *storage.YAMLStore
	activeTab    tab
	tasksView    views.TasksView
	activityView views.ActivityView
	agendaView   views.AgendaView
	searchView   views.SearchView
	futureView   views.FutureView
	filterView   views.FilterView
	filterMode   bool
	width        int
	height       int
}

func NewApp(store *storage.YAMLStore, contexts ...string) App {
	now := todayDate()
	return App{
		store:        store,
		activeTab:    tabTasks,
		tasksView:    views.NewTasksView(store, now, contexts...),
		activityView: views.NewActivityView(store, now),
		agendaView:   views.NewAgendaView(store),
		searchView:   views.NewSearchView(store),
		futureView:   views.NewFutureView(store, contexts...),
		filterView:   views.NewFilterView(),
	}
}

func (a App) Init() tea.Cmd {
	return tea.Batch(
		a.tasksView.Load(),
		a.activityView.Load(),
		a.agendaView.Load(),
		a.futureView.Load(),
	)
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		return a, nil

	case tea.KeyMsg:
		if a.filterMode {
			var cmd tea.Cmd
			a.filterView, cmd = a.filterView.Update(msg)
			if !a.filterView.IsActive() {
				a.filterMode = false
			}
			return a, cmd
		}
		return a.handleKey(msg)

	case views.FilterAppliedMsg:
		var cmd tea.Cmd
		a.tasksView, cmd = a.tasksView.SetFilter(msg.Filter)
		a.filterMode = false
		return a, cmd

	case views.AgendaNavigateMsg:
		a.activeTab = tabTasks
		a.tasksView = views.NewTasksView(a.store, msg.Date, a.tasksView.Contexts()...)
		return a, a.tasksView.Load()

	case views.SearchNavigateTaskMsg:
		a.activeTab = tabTasks
		a.tasksView = views.NewTasksView(a.store, msg.Date, a.tasksView.Contexts()...)
		return a, a.tasksView.Load()

	case views.SearchNavigateActivityMsg:
		a.activeTab = tabActivity
		a.activityView = views.NewActivityView(a.store, msg.Date)
		return a, a.activityView.Load()

	case views.SearchResultsMsg:
		var cmd tea.Cmd
		a.searchView, cmd = a.searchView.Update(msg)
		return a, cmd

	case views.AgendaLoadedMsg:
		var cmd tea.Cmd
		a.agendaView, cmd = a.agendaView.Update(msg)
		return a, cmd

	case views.TasksMsg:
		var cmd tea.Cmd
		a.tasksView, cmd = a.tasksView.Update(msg)
		// Reload agenda when tasks change
		return a, tea.Batch(cmd, a.agendaView.Load())

	case views.ActivityMsg:
		var cmd tea.Cmd
		a.activityView, cmd = a.activityView.Update(msg)
		return a, cmd

	case views.FutureMsg:
		var cmd tea.Cmd
		a.futureView, cmd = a.futureView.Update(msg)
		return a, cmd

	case views.FutureLinkedActivitiesMsg:
		var cmd tea.Cmd
		a.futureView, cmd = a.futureView.Update(msg)
		return a, cmd
	}

	// Route blink and other internal messages to filter when active
	if a.filterMode {
		var cmd tea.Cmd
		a.filterView, cmd = a.filterView.Update(msg)
		return a, cmd
	}

	// Delegate to active view
	var cmd tea.Cmd
	switch a.activeTab {
	case tabTasks:
		a.tasksView, cmd = a.tasksView.Update(msg)
	case tabActivity:
		a.activityView, cmd = a.activityView.Update(msg)
	case tabAgenda:
		a.agendaView, cmd = a.agendaView.Update(msg)
	case tabSearch:
		a.searchView, cmd = a.searchView.Update(msg)
	case tabFuture:
		a.futureView, cmd = a.futureView.Update(msg)
	}
	return a, cmd
}

func (a App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// When a form/input is active in any view, pass all keys through
	// without intercepting tab-switch shortcuts.
	inputActive := (a.activeTab == tabTasks && a.tasksView.IsInputActive()) ||
		(a.activeTab == tabActivity && a.activityView.IsInputActive()) ||
		(a.activeTab == tabSearch && a.searchView.IsInputActive()) ||
		(a.activeTab == tabFuture && a.futureView.IsInputActive())

	if !inputActive {
		switch msg.String() {
		case "q", "ctrl+c":
			return a, tea.Quit
		case "1":
			a.activeTab = tabTasks
			return a, nil
		case "2":
			a.activeTab = tabActivity
			return a, nil
		case "3":
			a.activeTab = tabAgenda
			return a, a.agendaView.Load()
		case "4":
			a.activeTab = tabSearch
			a.searchView = views.NewSearchView(a.store)
			return a, nil
		case "5":
			a.activeTab = tabFuture
			return a, a.futureView.Load()
		case "/":
			if a.activeTab == tabTasks {
				a.filterMode = true
				blinkCmd := a.filterView.Activate()
				return a, blinkCmd
			}
		}
	}

	// Allow ctrl+c to quit even inside forms, but not q (it should type normally)
	if inputActive && msg.String() == "ctrl+c" {
		return a, tea.Quit
	}

	var cmd tea.Cmd
	switch a.activeTab {
	case tabTasks:
		a.tasksView, cmd = a.tasksView.Update(msg)
	case tabActivity:
		a.activityView, cmd = a.activityView.Update(msg)
	case tabAgenda:
		a.agendaView, cmd = a.agendaView.Update(msg)
	case tabSearch:
		a.searchView, cmd = a.searchView.Update(msg)
	case tabFuture:
		a.futureView, cmd = a.futureView.Update(msg)
	}
	return a, cmd
}

func (a App) View() string {
	if a.width == 0 {
		return "Loading…"
	}

	header := a.renderHeader()
	body := a.renderBody()
	statusBar := a.renderStatusBar()

	return lipgloss.JoinVertical(lipgloss.Left,
		header,
		body,
		statusBar,
	)
}

func (a App) renderHeader() string {
	tabStyle := func(t tab, label string) string {
		if a.activeTab == t {
			return lipgloss.NewStyle().
				Bold(true).
				Foreground(lipgloss.Color("63")).
				Underline(true).
				Padding(0, 1).
				Render(label)
		}
		return lipgloss.NewStyle().
			Foreground(lipgloss.Color("240")).
			Padding(0, 1).
			Render(label)
	}

	tabs := lipgloss.JoinHorizontal(lipgloss.Top,
		tabStyle(tabTasks, "1 Tasks"),
		tabStyle(tabActivity, "2 Activity"),
		tabStyle(tabAgenda, "3 Agenda"),
		tabStyle(tabSearch, "4 Search"),
		tabStyle(tabFuture, "5 Future"),
	)

	date := lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Render(time.Now().Format("Mon 2006-01-02  15:04"))

	gap := strings.Repeat(" ", max(0, a.width-lipgloss.Width(tabs)-lipgloss.Width(date)-2))

	return lipgloss.NewStyle().
		Background(lipgloss.Color("235")).
		Width(a.width).
		Render(tabs + gap + date)
}

func (a App) renderBody() string {
	bodyHeight := a.height - 3

	if a.filterMode {
		overlay := a.filterView.View()
		base := a.tasksView.View(a.width, bodyHeight)
		ow := lipgloss.Width(overlay)
		oh := lipgloss.Height(overlay)
		ox := (a.width - ow) / 2
		oy := (bodyHeight - oh) / 2
		return placeOverlay(base, overlay, ox, oy, a.width, bodyHeight)
	}

	switch a.activeTab {
	case tabTasks:
		return a.tasksView.View(a.width, bodyHeight)
	case tabActivity:
		return a.activityView.View(a.width, bodyHeight)
	case tabAgenda:
		return a.agendaView.View(a.width, bodyHeight)
	case tabSearch:
		return a.searchView.View(a.width, bodyHeight)
	case tabFuture:
		return a.futureView.View(a.width, bodyHeight)
	}
	return ""
}

func (a App) renderStatusBar() string {
	var hints string
	switch a.activeTab {
	case tabTasks:
		hints = "a add  e edit  d done  c carry  n notes  L log activity  D delete  / filter  h/l day  tab detail  1-5 tabs  q quit"
	case tabActivity:
		hints = "a add  D delete  j/k navigate  h/l day  1-5 tabs  q quit"
	case tabAgenda:
		hints = "j/k navigate  d done  enter → Tasks  r refresh  1-5 tabs  q quit"
	case tabSearch:
		if a.searchView.IsInputActive() {
			hints = "type to search  ↓/enter → navigate results  esc → stop typing  q quit  1-5 tabs"
		} else {
			hints = "j/k navigate  enter → jump  i → edit search  q quit  1-5 tabs"
		}
	case tabFuture:
		hints = "a add  e edit  d done  s schedule  n notes  L log  D delete  tab detail  1-5 tabs  q quit"
	}
	return lipgloss.NewStyle().
		Foreground(lipgloss.Color("240")).
		Width(a.width).
		Padding(0, 1).
		Render(hints)
}

func placeOverlay(base, overlay string, ox, oy, width, height int) string {
	baseLines := strings.Split(base, "\n")
	overlayLines := strings.Split(overlay, "\n")

	for i, ol := range overlayLines {
		row := oy + i
		if row < 0 || row >= len(baseLines) {
			continue
		}
		bl := baseLines[row]
		blRunes := []rune(stripANSI(bl))

		prefix := ""
		if ox > 0 && ox <= len(blRunes) {
			prefix = string(blRunes[:ox])
		}
		baseLines[row] = prefix + ol
	}
	return strings.Join(baseLines, "\n")
}

func stripANSI(s string) string {
	var result strings.Builder
	inEsc := false
	for _, r := range s {
		if r == '\x1b' {
			inEsc = true
			continue
		}
		if inEsc {
			if r == 'm' {
				inEsc = false
			}
			continue
		}
		result.WriteRune(r)
	}
	return result.String()
}

func todayDate() time.Time {
	t := time.Now()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func Run(store *storage.YAMLStore, contexts ...string) error {
	app := NewApp(store, contexts...)
	p := tea.NewProgram(app, tea.WithAltScreen())
	_, err := p.Run()
	if err != nil {
		return fmt.Errorf("tui: %w", err)
	}
	return nil
}
