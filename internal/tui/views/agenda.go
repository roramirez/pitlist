package views

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/roramirez/pitlist/internal/model"
	"github.com/roramirez/pitlist/internal/storage"
)

type AgendaNavigateMsg struct{ Date time.Time }
type AgendaTaskDoneMsg struct {
	TaskID string
	Date   time.Time
}

type agendaItem struct {
	task *model.Task
	date time.Time
}

type AgendaView struct {
	store  *storage.YAMLStore
	items  []agendaItem // flattened list of (date, task) pairs
	cursor int
	width  int
	height int
	scroll int
}

func NewAgendaView(store *storage.YAMLStore) AgendaView {
	return AgendaView{store: store}
}

func agendaToday() time.Time {
	t := time.Now()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func (v AgendaView) Load() tea.Cmd {
	return func() tea.Msg {
		return v.loadItems()
	}
}

type AgendaLoadedMsg struct{ items []agendaItem }

func (v AgendaView) loadItems() tea.Msg {
	today := agendaToday()
	start := today.AddDate(0, 0, -7)
	end := today.AddDate(0, 0, 7)

	var items []agendaItem
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		plan, err := v.store.GetDayPlan(d)
		if err != nil {
			continue
		}
		for i := range plan.Tasks {
			t := &plan.Tasks[i]
			if t.Status == model.StatusDone || t.Status == model.StatusCancelled {
				continue
			}
			dc := d
			tc := *t
			items = append(items, agendaItem{task: &tc, date: dc})
		}
	}
	return AgendaLoadedMsg{items: items}
}

func (v AgendaView) Update(msg tea.Msg) (AgendaView, tea.Cmd) {
	switch msg := msg.(type) {
	case AgendaLoadedMsg:
		v.items = msg.items
		if v.cursor >= len(v.items) {
			v.cursor = max(0, len(v.items)-1)
		}
		return v, nil

	case tea.KeyMsg:
		return v.handleKey(msg)

	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
	}
	return v, nil
}

func (v AgendaView) handleKey(msg tea.KeyMsg) (AgendaView, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if v.cursor < len(v.items)-1 {
			v.cursor++
			v.adjustScroll()
		}
	case "k", "up":
		if v.cursor > 0 {
			v.cursor--
			v.adjustScroll()
		}
	case "d":
		if len(v.items) > 0 {
			item := v.items[v.cursor]
			return v, v.markDone(item)
		}
	case "enter", "l", "right":
		if len(v.items) > 0 {
			return v, func() tea.Msg {
				return AgendaNavigateMsg{Date: v.items[v.cursor].date}
			}
		}
	case "r":
		return v, v.Load()
	}
	return v, nil
}

func (v AgendaView) markDone(item agendaItem) tea.Cmd {
	return func() tea.Msg {
		plan, err := v.store.GetDayPlan(item.date)
		if err != nil {
			return errMsg{err}
		}
		now := time.Now().UTC()
		for i := range plan.Tasks {
			if plan.Tasks[i].ID == item.task.ID {
				plan.Tasks[i].Status = model.StatusDone
				plan.Tasks[i].DoneAt = &now
				plan.Tasks[i].UpdatedAt = now
				break
			}
		}
		if err := v.store.SaveDayPlan(plan); err != nil {
			return errMsg{err}
		}
		// Reload after marking done
		av := AgendaView{store: v.store}
		return av.loadItems()
	}
}

func (v *AgendaView) adjustScroll() {
	visibleLines := v.height - 4
	if visibleLines < 1 {
		visibleLines = 10
	}
	if v.cursor < v.scroll {
		v.scroll = v.cursor
	}
	if v.cursor >= v.scroll+visibleLines {
		v.scroll = v.cursor - visibleLines + 1
	}
}

func (v AgendaView) View(width, height int) string {
	v.width = width
	v.height = height

	today := agendaToday()

	var lines []string
	lines = append(lines, sTitle.Render("Agenda  — 7 days back · today · 7 days ahead"), "")

	if len(v.items) == 0 {
		lines = append(lines, sMuted.Render("  Nothing pending. Clear agenda!"))
	} else {
		visibleLines := height - 6
		if visibleLines < 1 {
			visibleLines = 20
		}

		var prevDate time.Time
		itemIdx := -1

		for i, item := range v.items {
			itemIdx++

			// Day header whenever date changes
			if !item.date.Equal(prevDate) {
				prevDate = item.date
				lines = append(lines, v.renderDayHeader(item.date, today))
			}

			if itemIdx < v.scroll || itemIdx >= v.scroll+visibleLines {
				continue
			}

			selected := i == v.cursor
			lines = append(lines, renderAgendaTask(item.task, selected))
		}

		// Scroll indicator
		if len(v.items) > visibleLines {
			lines = append(lines, "", sMuted.Render(fmt.Sprintf(
				"  %d/%d  j/k navigate", v.cursor+1, len(v.items),
			)))
		}
	}

	content := strings.Join(lines, "\n")
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Width(width-2).
		Height(height-2).
		Padding(0, 1).
		Render(content)
}

func (v AgendaView) renderDayHeader(d, today time.Time) string {
	label := d.Format("Mon Jan 02")
	var tag string
	switch {
	case d.Equal(today):
		tag = sAccent.Bold(true).Render("  ← today")
	case d.Before(today):
		tag = sHigh.Render("  overdue")
	case d.Equal(today.AddDate(0, 0, 1)):
		tag = sCarried.Render("  tomorrow")
	}
	header := sTitle.Render(label) + tag
	sep := sMuted.Render(strings.Repeat("─", 36))
	return "\n" + sep + "\n" + "  " + header
}

func renderAgendaTask(t *model.Task, selected bool) string {
	check := "[ ]"
	if t.Status == model.StatusInProgress {
		check = "[~]"
	}

	priority := ""
	if t.Priority == model.PriorityHigh {
		priority = sHigh.Render(" !")
	}

	carry := ""
	if t.CarryFrom != "" {
		carry = sCarried.Render(" ↑")
	}

	labels := ""
	if len(t.Labels) > 0 {
		labels = sAccent.Render(" [" + strings.Join(t.Labels, ", ") + "]")
	}

	line := fmt.Sprintf("    %s %s%s%s%s", check, t.Title, priority, carry, labels)
	if selected {
		line = sSelected.Render(line)
	}
	return line
}
