package views

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/roramirez/pitlist/internal/model"
	"github.com/roramirez/pitlist/internal/storage"
)

type ActivityMsg struct{ Log *model.ActivityLog }

const actFormFields = 5 // desc, tags, duration, date, taskRef

type addForm struct {
	active      bool
	focusIdx    int
	description textinput.Model
	tags        textinput.Model
	duration    textinput.Model
	dateInput   textinput.Model
	taskRef     textinput.Model
}

func newAddForm() addForm {
	desc := textinput.New()
	desc.Placeholder = "What did you do?"
	desc.Focus()

	tags := textinput.New()
	tags.Placeholder = "work debugging (space-separated)"

	dur := textinput.New()
	dur.Placeholder = "minutes (optional)"
	dur.CharLimit = 4

	di := textinput.New()
	di.CharLimit = 16
	di.SetValue(time.Now().Format("2006-01-02T15:04"))

	ref := textinput.New()
	ref.Placeholder = "task ID (optional)"
	ref.CharLimit = 20

	return addForm{description: desc, tags: tags, duration: dur, dateInput: di, taskRef: ref}
}

type ActivityView struct {
	store  *storage.YAMLStore
	date   time.Time
	log    *model.ActivityLog
	cursor int
	form   addForm
	width  int
	height int
}

func (v ActivityView) IsInputActive() bool { return v.form.active }

func NewActivityView(store *storage.YAMLStore, date time.Time) ActivityView {
	return ActivityView{
		store: store,
		date:  date,
		log:   &model.ActivityLog{Date: date, Entries: []model.ActivityEntry{}},
		form:  newAddForm(),
	}
}

func (v ActivityView) Load() tea.Cmd {
	return func() tea.Msg {
		log, err := v.store.GetActivityLog(v.date)
		if err != nil {
			return errMsg{err}
		}
		return ActivityMsg{log}
	}
}

func (v ActivityView) Update(msg tea.Msg) (ActivityView, tea.Cmd) {
	switch msg := msg.(type) {
	case ActivityMsg:
		v.log = msg.Log
		return v, nil

	case tea.KeyMsg:
		if v.form.active {
			return v.updateForm(msg)
		}
		return v.updateNormal(msg)

	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
	}
	return v, nil
}

func (v ActivityView) updateNormal(msg tea.KeyMsg) (ActivityView, tea.Cmd) {
	switch msg.String() {
	case "j", "down":
		if v.cursor < len(v.log.Entries)-1 {
			v.cursor++
		}
	case "k", "up":
		if v.cursor > 0 {
			v.cursor--
		}
	case "h", "left", "[":
		v.date = v.date.AddDate(0, 0, -1)
		v.cursor = 0
		return v, v.Load()
	case "l", "right", "]":
		v.date = v.date.AddDate(0, 0, 1)
		v.cursor = 0
		return v, v.Load()
	case "a":
		v.form = newAddForm()
		v.form.active = true
		return v, textinput.Blink
	case "D":
		if v.log != nil && len(v.log.Entries) > 0 {
			return v, v.deleteEntry(v.cursor)
		}
	}
	return v, nil
}

func (v ActivityView) deleteEntry(idx int) tea.Cmd {
	date := v.date
	entries := make([]model.ActivityEntry, len(v.log.Entries))
	copy(entries, v.log.Entries)

	return func() tea.Msg {
		log, err := v.store.GetActivityLog(date)
		if err != nil {
			return errMsg{err}
		}
		if idx < 0 || idx >= len(log.Entries) {
			return ActivityMsg{log}
		}
		log.Entries = append(log.Entries[:idx], log.Entries[idx+1:]...)
		if err := v.store.SaveActivityLog(log); err != nil {
			return errMsg{err}
		}
		return ActivityMsg{log}
	}
}

func (v ActivityView) updateForm(msg tea.KeyMsg) (ActivityView, tea.Cmd) {
	switch msg.String() {
	case "tab", "down":
		v.form.focusIdx = (v.form.focusIdx + 1) % actFormFields
		return v.focusField()
	case "shift+tab", "up":
		v.form.focusIdx = (v.form.focusIdx + actFormFields - 1) % actFormFields
		return v.focusField()
	case "enter":
		if v.form.focusIdx == actFormFields-1 {
			v.form.active = false
			return v, v.submitForm()
		}
		v.form.focusIdx = (v.form.focusIdx + 1) % actFormFields
		return v.focusField()
	case "ctrl+s":
		v.form.active = false
		return v, v.submitForm()
	case "esc":
		v.form.active = false
		return v, nil
	default:
		var cmd tea.Cmd
		switch v.form.focusIdx {
		case 0:
			v.form.description, cmd = v.form.description.Update(msg)
		case 1:
			v.form.tags, cmd = v.form.tags.Update(msg)
		case 2:
			v.form.duration, cmd = v.form.duration.Update(msg)
		case 3:
			v.form.dateInput, cmd = v.form.dateInput.Update(msg)
		case 4:
			v.form.taskRef, cmd = v.form.taskRef.Update(msg)
		}
		return v, cmd
	}
}

func (v ActivityView) focusField() (ActivityView, tea.Cmd) {
	v.form.description.Blur()
	v.form.tags.Blur()
	v.form.duration.Blur()
	v.form.dateInput.Blur()
	v.form.taskRef.Blur()
	// Auto-compute date when focusing it
	if v.form.focusIdx == 3 {
		dur := 0
		if d, err := strconv.Atoi(strings.TrimSpace(v.form.duration.Value())); err == nil {
			dur = d
		}
		ts := time.Now().Add(-time.Duration(dur) * time.Minute)
		v.form.dateInput.SetValue(ts.Format("2006-01-02T15:04"))
	}
	switch v.form.focusIdx {
	case 0:
		v.form.description.Focus()
	case 1:
		v.form.tags.Focus()
	case 2:
		v.form.duration.Focus()
	case 3:
		v.form.dateInput.Focus()
	case 4:
		v.form.taskRef.Focus()
	}
	return v, textinput.Blink
}

func (v ActivityView) submitForm() tea.Cmd {
	desc := strings.TrimSpace(v.form.description.Value())
	if desc == "" {
		return nil
	}

	var tags []string
	for _, t := range strings.Fields(v.form.tags.Value()) {
		tags = append(tags, t)
	}

	dur := 0
	if d, err := strconv.Atoi(strings.TrimSpace(v.form.duration.Value())); err == nil {
		dur = d
	}

	ref := strings.TrimSpace(v.form.taskRef.Value())

	ts := time.Now().Add(-time.Duration(dur) * time.Minute)
	if raw := strings.TrimSpace(v.form.dateInput.Value()); raw != "" {
		if t, err := time.ParseInLocation("2006-01-02T15:04", raw, time.Local); err == nil {
			ts = t
		}
	}
	entryDate := time.Date(ts.Year(), ts.Month(), ts.Day(), 0, 0, 0, 0, time.UTC)
	tsUTC := ts.UTC()

	return func() tea.Msg {
		log, err := v.store.GetActivityLog(entryDate)
		if err != nil {
			return errMsg{err}
		}

		entry := model.ActivityEntry{
			ID:          storage.NextActivityID(log),
			Timestamp:   tsUTC,
			Description: desc,
			Tags:        tags,
			TaskRef:     ref,
			DurationMin: dur,
		}
		log.Entries = append(log.Entries, entry)
		if err := v.store.SaveActivityLog(log); err != nil {
			return errMsg{err}
		}
		if ref != "" {
			_ = v.store.AddActivityRefToTask(ref, model.ActivityRef{
				ID:   entry.ID,
				Date: entryDate.Format("2006-01-02"),
			})
		}
		return ActivityMsg{log}
	}
}

func (v ActivityView) View(width, height int) string {
	muted := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	bold := lipgloss.NewStyle().Bold(true)

	header := muted.Render("←") + "  " + bold.Render("Activity Log  "+v.date.Format("Mon Jan 02 2006")) + "  " + muted.Render("→")
	var lines []string
	lines = append(lines, header, "")

	if v.form.active {
		lines = append(lines, v.renderForm()...)
		lines = append(lines, "")
	}

	if v.log == nil || len(v.log.Entries) == 0 {
		lines = append(lines, muted.Render("No entries. Press 'a' to add one."))
	} else {
		var totalMin int
		for i, e := range v.log.Entries {
			lines = append(lines, renderEntryLine(e, i == v.cursor))
			totalMin += e.DurationMin
		}
		if totalMin > 0 {
			lines = append(lines, "")
			lines = append(lines, muted.Render(fmt.Sprintf("Total logged: %dh %dm", totalMin/60, totalMin%60)))
		}
	}

	content := strings.Join(lines, "\n")
	return lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("63")).
		Width(width - 2).
		Height(height - 2).
		Padding(0, 1).
		Render(content)
}

func (v ActivityView) renderForm() []string {
	bold := lipgloss.NewStyle().Bold(true)
	muted := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	label := func(idx int, s string) string {
		if v.form.focusIdx == idx {
			return bold.Render("> " + s)
		}
		return muted.Render("  " + s)
	}

	return []string{
		bold.Render("─── New Activity ───"),
		label(0, "Description:") + "  " + v.form.description.View(),
		label(1, "Tags:       ") + "  " + v.form.tags.View(),
		label(2, "Duration:   ") + "  " + v.form.duration.View(),
		label(3, "Date:       ") + "  " + v.form.dateInput.View(),
		label(4, "Task ref:   ") + "  " + v.form.taskRef.View(),
		muted.Render("  tab next  ctrl+s save  esc cancel"),
		strings.Repeat("─", 40),
	}
}

func renderEntryLine(e model.ActivityEntry, selected bool) string {
	timeStr := e.Timestamp.Local().Format("15:04")

	dur := ""
	if e.DurationMin > 0 {
		dur = fmt.Sprintf(" %dm", e.DurationMin)
	}

	tags := ""
	if len(e.Tags) > 0 {
		tags = lipgloss.NewStyle().Foreground(lipgloss.Color("63")).Render(" [" + strings.Join(e.Tags, ", ") + "]")
	}

	ref := ""
	if e.TaskRef != "" {
		ref = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(" → " + e.TaskRef)
	}

	line := fmt.Sprintf("  %s%s  %s%s%s",
		lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(timeStr),
		lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render(dur),
		e.Description,
		tags,
		ref,
	)

	if selected {
		line = lipgloss.NewStyle().Background(lipgloss.Color("236")).Render(line)
	}
	return line
}
