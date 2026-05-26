package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/roramirez/pitlist/internal/model"
	"github.com/roramirez/pitlist/internal/storage"
)

type FutureMsg struct {
	List *model.FutureList
}

type FutureLinkedActivitiesMsg struct{ Entries []model.ActivityEntry }

type futureDetailMode int

const (
	futureDetailNormal futureDetailMode = iota
	futureDetailEditNotes
	futureDetailLogActivity
	futureDetailEditTask
	futureDetailSchedule
)

type FutureView struct {
	store            *storage.YAMLStore
	list             *model.FutureList
	linkedActivities []model.ActivityEntry
	contexts         []string
	cursor           int
	pane             int // 0=list, 1=detail
	adding           bool
	tForm            taskForm
	detailMode       futureDetailMode
	notesArea        textarea.Model
	logForm          quickLogForm
	scheduleInput    textinput.Model
	scheduleTaskID   string
	width            int
	height           int
}

func NewFutureView(store *storage.YAMLStore, contexts ...string) FutureView {
	ta := textarea.New()
	ta.Placeholder = "Task notes…"
	ta.ShowLineNumbers = false

	si := textinput.New()
	si.Placeholder = "today, tomorrow, YYYY-MM-DD…"
	si.CharLimit = 20

	return FutureView{
		store:         store,
		list:          &model.FutureList{Tasks: []model.Task{}},
		contexts:      contexts,
		notesArea:     ta,
		scheduleInput: si,
	}
}

func (v FutureView) Load() tea.Cmd {
	return func() tea.Msg {
		list, err := v.store.GetFutureList()
		if err != nil {
			return errMsg{err}
		}
		return FutureMsg{List: list}
	}
}

func (v FutureView) loadMsg() tea.Msg {
	list, err := v.store.GetFutureList()
	if err != nil {
		return errMsg{err}
	}
	return FutureMsg{List: list}
}

func (v FutureView) loadLinkedActivities(taskID string) tea.Cmd {
	if taskID == "" {
		return nil
	}
	return func() tea.Msg {
		task, err := v.store.GetFutureTaskByID(taskID)
		if err != nil {
			return FutureLinkedActivitiesMsg{}
		}
		entries, err := v.store.GetActivitiesByRefs(task.ActivityRefs, time.Now())
		if err != nil {
			return errMsg{err}
		}
		result := make([]model.ActivityEntry, len(entries))
		for i, e := range entries {
			result[i] = *e
		}
		return FutureLinkedActivitiesMsg{Entries: result}
	}
}

func (v FutureView) Update(msg tea.Msg) (FutureView, tea.Cmd) {
	switch msg := msg.(type) {
	case FutureMsg:
		v.list = msg.List
		if v.cursor >= len(v.list.Tasks) {
			v.cursor = max(0, len(v.list.Tasks)-1)
		}
		if len(v.list.Tasks) > 0 {
			return v, v.loadLinkedActivities(v.list.Tasks[v.cursor].ID)
		}
		return v, nil

	case FutureLinkedActivitiesMsg:
		v.linkedActivities = msg.Entries
		return v, nil

	case tea.KeyMsg:
		if v.adding {
			return v.updateAdding(msg)
		}
		switch v.detailMode {
		case futureDetailEditNotes:
			return v.updateNotes(msg)
		case futureDetailLogActivity:
			return v.updateLogForm(msg)
		case futureDetailEditTask:
			return v.updateTaskForm(msg)
		case futureDetailSchedule:
			return v.updateSchedule(msg)
		}
		return v.updateNormal(msg)

	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
	}
	return v, nil
}

func (v FutureView) updateNormal(msg tea.KeyMsg) (FutureView, tea.Cmd) {
	tasks := v.list.Tasks
	switch msg.String() {
	case "j", "down":
		if v.cursor+1 < len(tasks) {
			v.cursor++
			return v, v.loadLinkedActivities(tasks[v.cursor].ID)
		}
	case "k", "up":
		if v.cursor > 0 {
			v.cursor--
			return v, v.loadLinkedActivities(tasks[v.cursor].ID)
		}
	case "tab":
		if v.pane == 0 {
			v.pane = 1
		} else {
			v.pane = 0
		}
	case "a":
		if v.pane == 0 {
			v.tForm = newTaskForm("", "", "", v.contexts, "", "medium")
			v.adding = true
			return v, textinput.Blink
		}
	case "d":
		if len(tasks) > 0 {
			return v, v.toggleDone(tasks[v.cursor].ID)
		}
	case "e":
		if len(tasks) > 0 {
			t := tasks[v.cursor]
			v.tForm = newTaskForm(t.ID, t.Title, t.Context, v.contexts, strings.Join(t.Labels, " "), string(t.Priority))
			v.detailMode = futureDetailEditTask
			v.pane = 1
			return v, textinput.Blink
		}
	case "n":
		if len(tasks) > 0 {
			v.pane = 1
			v.detailMode = futureDetailEditNotes
			v.notesArea.Reset()
			v.notesArea.SetValue(tasks[v.cursor].Notes)
			v.notesArea.Focus()
			return v, textarea.Blink
		}
	case "L":
		if len(tasks) > 0 {
			v.pane = 1
			v.detailMode = futureDetailLogActivity
			v.logForm = newQuickLogForm(tasks[v.cursor].ID)
			return v, textinput.Blink
		}
	case "s":
		if len(tasks) > 0 {
			v.scheduleTaskID = tasks[v.cursor].ID
			v.scheduleInput.Reset()
			v.scheduleInput.SetValue("today")
			v.scheduleInput.Focus()
			v.detailMode = futureDetailSchedule
			v.pane = 1
			return v, textinput.Blink
		}
	case "D":
		if len(tasks) > 0 {
			return v, v.deleteTask(tasks[v.cursor].ID)
		}
	}
	return v, nil
}

func (v FutureView) updateAdding(msg tea.KeyMsg) (FutureView, tea.Cmd) {
	switch msg.String() {
	case "tab", "down":
		v.tForm.focusIdx = (v.tForm.focusIdx + 1) % taskFormFields
		return v.focusTaskField()
	case "shift+tab", "up":
		v.tForm.focusIdx = (v.tForm.focusIdx + taskFormFields - 1) % taskFormFields
		return v.focusTaskField()
	case "ctrl+s":
		title := strings.TrimSpace(v.tForm.title.Value())
		if title == "" {
			v.adding = false
			return v, nil
		}
		v.adding = false
		return v, v.saveNewTask()
	case "enter":
		if v.tForm.focusIdx == taskFormFields-1 {
			title := strings.TrimSpace(v.tForm.title.Value())
			if title == "" {
				v.adding = false
				return v, nil
			}
			v.adding = false
			return v, v.saveNewTask()
		}
		v.tForm.focusIdx = (v.tForm.focusIdx + 1) % taskFormFields
		return v.focusTaskField()
	case "esc":
		v.adding = false
	default:
		v = v.handleFormKey(msg)
		return v, nil
	}
	return v, nil
}

func (v FutureView) updateTaskForm(msg tea.KeyMsg) (FutureView, tea.Cmd) {
	switch msg.String() {
	case "tab", "down":
		v.tForm.focusIdx = (v.tForm.focusIdx + 1) % taskFormFields
		return v.focusTaskField()
	case "shift+tab", "up":
		v.tForm.focusIdx = (v.tForm.focusIdx + taskFormFields - 1) % taskFormFields
		return v.focusTaskField()
	case "ctrl+s":
		v.detailMode = futureDetailNormal
		return v, v.saveEditTask()
	case "enter":
		if v.tForm.focusIdx == taskFormFields-1 {
			v.detailMode = futureDetailNormal
			return v, v.saveEditTask()
		}
		v.tForm.focusIdx = (v.tForm.focusIdx + 1) % taskFormFields
		return v.focusTaskField()
	case "esc":
		v.detailMode = futureDetailNormal
	default:
		v = v.handleFormKey(msg)
		return v, nil
	}
	return v, nil
}

func (v FutureView) handleFormKey(msg tea.KeyMsg) FutureView {
	n := len(v.tForm.contexts)
	switch v.tForm.focusIdx {
	case 0:
		v.tForm.title, _ = v.tForm.title.Update(msg)
	case 1:
		switch msg.String() {
		case "left", "h":
			if n > 0 {
				v.tForm.contextIdx = (v.tForm.contextIdx-1+n+1)%(n+1) - 1
				if v.tForm.contextIdx < -1 {
					v.tForm.contextIdx = n - 1
				}
			}
		case "right", "l":
			if n > 0 {
				v.tForm.contextIdx++
				if v.tForm.contextIdx >= n {
					v.tForm.contextIdx = -1
				}
			}
		}
	case 2:
		v.tForm.labels, _ = v.tForm.labels.Update(msg)
	case 3:
		v.tForm.priority, _ = v.tForm.priority.Update(msg)
	}
	return v
}

func (v FutureView) focusTaskField() (FutureView, tea.Cmd) {
	v.tForm.title.Blur()
	v.tForm.labels.Blur()
	v.tForm.priority.Blur()
	switch v.tForm.focusIdx {
	case 0:
		v.tForm.title.Focus()
		return v, textinput.Blink
	case 1:
		return v, nil
	case 2:
		v.tForm.labels.Focus()
		return v, textinput.Blink
	case 3:
		v.tForm.priority.Focus()
		return v, textinput.Blink
	}
	return v, nil
}

func (v FutureView) updateNotes(msg tea.KeyMsg) (FutureView, tea.Cmd) {
	switch msg.String() {
	case "esc":
		v.detailMode = futureDetailNormal
		v.notesArea.Blur()
		return v, nil
	case "ctrl+s":
		if len(v.list.Tasks) == 0 {
			v.detailMode = futureDetailNormal
			return v, nil
		}
		notes := v.notesArea.Value()
		id := v.list.Tasks[v.cursor].ID
		v.detailMode = futureDetailNormal
		v.notesArea.Blur()
		return v, v.saveNotes(id, notes)
	default:
		var cmd tea.Cmd
		v.notesArea, cmd = v.notesArea.Update(msg)
		return v, cmd
	}
}

func (v FutureView) updateLogForm(msg tea.KeyMsg) (FutureView, tea.Cmd) {
	switch msg.String() {
	case "esc":
		v.detailMode = futureDetailNormal
		return v, nil
	case "ctrl+s":
		cmd := v.submitLogForm()
		v.detailMode = futureDetailNormal
		return v, cmd
	case "tab", "down":
		v.logForm.focusIdx = (v.logForm.focusIdx + 1) % quickLogFields
		return v.focusLogField()
	case "shift+tab", "up":
		v.logForm.focusIdx = (v.logForm.focusIdx + quickLogFields - 1) % quickLogFields
		return v.focusLogField()
	case "enter":
		if v.logForm.focusIdx == quickLogFields-1 {
			v.detailMode = futureDetailNormal
			return v, v.submitLogForm()
		}
		v.logForm.focusIdx = (v.logForm.focusIdx + 1) % quickLogFields
		return v.focusLogField()
	default:
		var cmd tea.Cmd
		switch v.logForm.focusIdx {
		case 0:
			v.logForm.desc, cmd = v.logForm.desc.Update(msg)
		case 1:
			v.logForm.tags, cmd = v.logForm.tags.Update(msg)
		case 2:
			v.logForm.duration, cmd = v.logForm.duration.Update(msg)
		case 3:
			v.logForm.dateInput, cmd = v.logForm.dateInput.Update(msg)
		}
		return v, cmd
	}
}

func (v FutureView) focusLogField() (FutureView, tea.Cmd) {
	v.logForm.desc.Blur()
	v.logForm.tags.Blur()
	v.logForm.duration.Blur()
	v.logForm.dateInput.Blur()
	switch v.logForm.focusIdx {
	case 0:
		v.logForm.desc.Focus()
	case 1:
		v.logForm.tags.Focus()
	case 2:
		v.logForm.duration.Focus()
	case 3:
		v.logForm.dateInput.Focus()
	}
	return v, textinput.Blink
}

func (v FutureView) updateSchedule(msg tea.KeyMsg) (FutureView, tea.Cmd) {
	switch msg.String() {
	case "enter", "ctrl+s":
		raw := strings.TrimSpace(v.scheduleInput.Value())
		taskID := v.scheduleTaskID
		v.detailMode = futureDetailNormal
		v.scheduleInput.Blur()
		return v, v.scheduleTask(taskID, raw)
	case "esc":
		v.detailMode = futureDetailNormal
		v.scheduleInput.Blur()
	default:
		var cmd tea.Cmd
		v.scheduleInput, cmd = v.scheduleInput.Update(msg)
		return v, cmd
	}
	return v, nil
}

func (v FutureView) saveNewTask() tea.Cmd {
	title := strings.TrimSpace(v.tForm.title.Value())
	context := v.tForm.contextValue()
	labels := parseLabels(v.tForm.labels.Value())
	priority := parsePriority(v.tForm.priority.Value())
	return func() tea.Msg {
		list, err := v.store.GetFutureList()
		if err != nil {
			return errMsg{err}
		}
		now := time.Now().UTC()
		task := model.Task{
			ID:        storage.NextFutureTaskID(list),
			Title:     title,
			Context:   context,
			Labels:    labels,
			Status:    model.StatusTodo,
			Priority:  priority,
			CreatedAt: now,
			UpdatedAt: now,
		}
		list.Tasks = append(list.Tasks, task)
		if err := v.store.SaveFutureList(list); err != nil {
			return errMsg{err}
		}
		return v.loadMsg()
	}
}

func (v FutureView) saveEditTask() tea.Cmd {
	id := v.tForm.editID
	title := strings.TrimSpace(v.tForm.title.Value())
	context := v.tForm.contextValue()
	labels := parseLabels(v.tForm.labels.Value())
	priority := parsePriority(v.tForm.priority.Value())
	return func() tea.Msg {
		list, err := v.store.GetFutureList()
		if err != nil {
			return errMsg{err}
		}
		for i := range list.Tasks {
			if list.Tasks[i].ID == id {
				if title != "" {
					list.Tasks[i].Title = title
				}
				list.Tasks[i].Context = context
				list.Tasks[i].Labels = labels
				list.Tasks[i].Priority = priority
				list.Tasks[i].UpdatedAt = time.Now().UTC()
				break
			}
		}
		if err := v.store.SaveFutureList(list); err != nil {
			return errMsg{err}
		}
		return v.loadMsg()
	}
}

func (v FutureView) saveNotes(id, notes string) tea.Cmd {
	return func() tea.Msg {
		list, err := v.store.GetFutureList()
		if err != nil {
			return errMsg{err}
		}
		for i := range list.Tasks {
			if list.Tasks[i].ID == id {
				list.Tasks[i].Notes = notes
				list.Tasks[i].UpdatedAt = time.Now().UTC()
				break
			}
		}
		if err := v.store.SaveFutureList(list); err != nil {
			return errMsg{err}
		}
		return v.loadMsg()
	}
}

func (v FutureView) toggleDone(id string) tea.Cmd {
	return func() tea.Msg {
		list, err := v.store.GetFutureList()
		if err != nil {
			return errMsg{err}
		}
		now := time.Now().UTC()
		for i := range list.Tasks {
			if list.Tasks[i].ID == id {
				if list.Tasks[i].Status == model.StatusDone {
					list.Tasks[i].Status = model.StatusTodo
					list.Tasks[i].DoneAt = nil
				} else {
					list.Tasks[i].Status = model.StatusDone
					list.Tasks[i].DoneAt = &now
				}
				list.Tasks[i].UpdatedAt = now
				break
			}
		}
		if err := v.store.SaveFutureList(list); err != nil {
			return errMsg{err}
		}
		return v.loadMsg()
	}
}

func (v FutureView) deleteTask(id string) tea.Cmd {
	return func() tea.Msg {
		list, err := v.store.GetFutureList()
		if err != nil {
			return errMsg{err}
		}
		remaining := make([]model.Task, 0, len(list.Tasks))
		for _, t := range list.Tasks {
			if t.ID != id {
				remaining = append(remaining, t)
			}
		}
		list.Tasks = remaining
		if err := v.store.SaveFutureList(list); err != nil {
			return errMsg{err}
		}
		return v.loadMsg()
	}
}

func (v FutureView) submitLogForm() tea.Cmd {
	desc := strings.TrimSpace(v.logForm.desc.Value())
	if desc == "" {
		return nil
	}
	var tags []string
	for _, t := range strings.Fields(v.logForm.tags.Value()) {
		tags = append(tags, t)
	}
	dur := 0
	if raw := strings.TrimSpace(v.logForm.duration.Value()); raw != "" {
		var d int
		if n, _ := fmt.Sscanf(raw, "%d", &d); n == 1 {
			dur = d
		}
	}
	taskID := v.logForm.taskID

	now := time.Now()
	ts := now.Add(-time.Duration(dur) * time.Minute)
	if raw := strings.TrimSpace(v.logForm.dateInput.Value()); raw != "" {
		if t, err := time.ParseInLocation(model.DateTimeFormat, raw, time.Local); err == nil {
			ts = t
		}
	}
	entryDate := time.Date(ts.Year(), ts.Month(), ts.Day(), 0, 0, 0, 0, time.UTC)
	tsUTC := ts.UTC()

	return func() tea.Msg {
		actLog, err := v.store.GetActivityLog(entryDate)
		if err != nil {
			return errMsg{err}
		}
		entry := model.ActivityEntry{
			ID:          storage.NextActivityID(actLog),
			Timestamp:   tsUTC,
			Description: desc,
			Tags:        tags,
			TaskRef:     taskID,
			DurationMin: dur,
		}
		actLog.Entries = append(actLog.Entries, entry)
		if err := v.store.SaveActivityLog(actLog); err != nil {
			return errMsg{err}
		}
		if taskID != "" {
			_ = v.store.AddActivityRefToFutureTask(taskID, model.ActivityRef{
				ID:   entry.ID,
				Date: entryDate.Format(model.DateFormat),
			})
		}
		return v.loadMsg()
	}
}

func (v FutureView) scheduleTask(taskID, rawDate string) tea.Cmd {
	return func() tea.Msg {
		list, err := v.store.GetFutureList()
		if err != nil {
			return errMsg{err}
		}

		var task *model.Task
		remaining := make([]model.Task, 0, len(list.Tasks))
		for i := range list.Tasks {
			if list.Tasks[i].ID == taskID {
				t := list.Tasks[i]
				task = &t
			} else {
				remaining = append(remaining, list.Tasks[i])
			}
		}
		if task == nil {
			return v.loadMsg()
		}

		// Parse date keyword or YYYY-MM-DD
		day := time.Now()
		switch strings.ToLower(rawDate) {
		case "", "today":
			d := time.Now()
			day = time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC)
		case "tomorrow":
			d := time.Now()
			day = time.Date(d.Year(), d.Month(), d.Day(), 0, 0, 0, 0, time.UTC).AddDate(0, 0, 1)
		default:
			if parsed, err := time.Parse(model.DateFormat, rawDate); err == nil {
				day = parsed
			}
		}

		list.Tasks = remaining
		if err := v.store.SaveFutureList(list); err != nil {
			return errMsg{err}
		}

		plan, err := v.store.GetDayPlan(day)
		if err != nil {
			return errMsg{err}
		}
		moved := *task
		moved.ID = storage.NextTaskID(plan)
		moved.UpdatedAt = time.Now().UTC()
		plan.Tasks = append(plan.Tasks, moved)
		if err := v.store.SaveDayPlan(plan); err != nil {
			return errMsg{err}
		}

		return v.loadMsg()
	}
}

func (v FutureView) IsInputActive() bool {
	return v.adding || v.detailMode != futureDetailNormal
}

func (v FutureView) View(width, height int) string {
	listWidth := width / 2
	detailWidth := width - listWidth - 3

	listContent := v.renderList(listWidth - 4)
	detailContent := v.renderDetail(detailWidth - 4)

	listStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("238")).
		Width(listWidth).
		Height(height-2).
		Padding(0, 1)

	detailStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("238")).
		Width(detailWidth).
		Height(height-2).
		Padding(0, 1)

	if v.pane == 0 {
		listStyle = listStyle.BorderForeground(lipgloss.Color("63"))
	} else {
		detailStyle = detailStyle.BorderForeground(lipgloss.Color("63"))
	}

	return lipgloss.JoinHorizontal(lipgloss.Top,
		listStyle.Render(listContent),
		detailStyle.Render(detailContent),
	)
}

func (v FutureView) renderList(width int) string {
	header := sTitle.Render("Future / Backlog")
	var lines []string
	lines = append(lines, header, "")

	if v.adding {
		lines = append(lines, v.renderTaskFormInline()...)
	}

	tasks := v.list.Tasks
	if len(tasks) == 0 && !v.adding {
		lines = append(lines, sMuted.Render("  No future tasks. Press 'a' to add one."))
	} else {
		for i, t := range tasks {
			lines = append(lines, renderTaskLine(t, i == v.cursor, width))
		}
	}

	return strings.Join(lines, "\n")
}

func (v FutureView) renderTaskFormInline() []string {
	fl := func(idx int, label string) string {
		if v.tForm.focusIdx == idx {
			return sTitle.Render("> " + label)
		}
		return sMuted.Render("  " + label)
	}
	return []string{
		sTitle.Render("── New future task ──"),
		fl(0, "Title:    ") + " " + v.tForm.title.View(),
		fl(1, "Context:  ") + " " + v.tForm.contextDisplay(v.tForm.focusIdx == 1),
		fl(2, "Labels:   ") + " " + v.tForm.labels.View(),
		fl(3, "Priority: ") + " " + v.tForm.priority.View(),
		sMuted.Render("  tab next  ←/→ context  ctrl+s save  esc cancel"),
		"",
	}
}

func (v FutureView) renderDetail(width int) string {
	tasks := v.list.Tasks
	if len(tasks) == 0 {
		return sMuted.Render("No task selected.")
	}
	if v.cursor >= len(tasks) {
		return ""
	}
	t := tasks[v.cursor]

	switch v.detailMode {
	case futureDetailEditNotes:
		return v.renderNotesEditor(t, width)
	case futureDetailLogActivity:
		return v.renderLogForm(t, width)
	case futureDetailEditTask:
		return v.renderTaskFormDetail(t, width)
	case futureDetailSchedule:
		return v.renderSchedulePrompt(t, width)
	default:
		return v.renderTaskDetail(t, width)
	}
}

func (v FutureView) renderTaskDetail(t model.Task, width int) string {
	var lines []string
	lines = append(lines, sTitle.Render(t.Title))
	lines = append(lines, strings.Repeat("─", min(len(t.Title)+2, width)))
	lines = append(lines, "")
	if t.Context != "" {
		lines = append(lines, fmt.Sprintf("Context:  %s", t.Context))
	}
	lines = append(lines, fmt.Sprintf("Status:   %s", t.Status))
	lines = append(lines, fmt.Sprintf("Priority: %s", t.Priority))
	if len(t.Labels) > 0 {
		lines = append(lines, fmt.Sprintf("Labels:   %s", strings.Join(t.Labels, "  ")))
	}

	if t.Notes != "" {
		lines = append(lines, "", sTitle.Render("Notes:"), t.Notes)
	}

	linked := v.linkedActivities
	if len(linked) > 0 {
		totalMin := 0
		for _, e := range linked {
			totalMin += e.DurationMin
		}
		header := sTitle.Render("Activity:")
		if totalMin > 0 {
			total := fmt.Sprintf("%dh %02dm", totalMin/60, totalMin%60)
			header += "  " + sCarried.Render("∑ "+total)
		}
		lines = append(lines, "", header)
		for _, e := range linked {
			dur := ""
			if e.DurationMin > 0 {
				dur = fmt.Sprintf(" %dm", e.DurationMin)
			}
			tags := ""
			if len(e.Tags) > 0 {
				tags = sMuted.Render(" [" + strings.Join(e.Tags, ", ") + "]")
			}
			lines = append(lines, fmt.Sprintf("  %s%s  %s%s",
				sMuted.Render(e.Timestamp.Local().Format("Jan 02 15:04")),
				sCarried.Render(dur),
				e.Description,
				tags,
			))
		}
	}

	lines = append(lines, "", sMuted.Render("ID: "+t.ID))
	lines = append(lines, "", sMuted.Render("n notes  L log  d done  s schedule  e edit  D delete  tab ←list"))
	return strings.Join(lines, "\n")
}

func (v FutureView) renderTaskFormDetail(t model.Task, width int) string {
	fl := func(idx int, label string) string {
		if v.tForm.focusIdx == idx {
			return sTitle.Render("> " + label)
		}
		return sMuted.Render("  " + label)
	}
	return strings.Join([]string{
		sTitle.Render("Edit task"),
		sMuted.Render("ID: " + t.ID),
		strings.Repeat("─", min(width, 36)),
		"",
		fl(0, "Title:    ") + " " + v.tForm.title.View(),
		fl(1, "Context:  ") + " " + v.tForm.contextDisplay(v.tForm.focusIdx == 1),
		fl(2, "Labels:   ") + " " + v.tForm.labels.View(),
		fl(3, "Priority: ") + " " + v.tForm.priority.View(),
		"",
		sMuted.Render("  tab next  ←/→ context  ctrl+s save  esc cancel"),
	}, "\n")
}

func (v FutureView) renderNotesEditor(t model.Task, width int) string {
	v.notesArea.SetWidth(width - 2)
	v.notesArea.SetHeight(10)
	return strings.Join([]string{
		sTitle.Render("Notes: " + t.Title),
		strings.Repeat("─", min(len(t.Title)+8, width)),
		"",
		v.notesArea.View(),
		"",
		sMuted.Render("ctrl+s save  esc cancel"),
	}, "\n")
}

func (v FutureView) renderLogForm(t model.Task, width int) string {
	fieldLabel := func(idx int, label string) string {
		if v.logForm.focusIdx == idx {
			return sTitle.Render("> " + label)
		}
		return sMuted.Render("  " + label)
	}
	return strings.Join([]string{
		sTitle.Render("Log activity"),
		sMuted.Render("→ " + t.Title),
		strings.Repeat("─", min(width, 36)),
		"",
		fieldLabel(0, "Description: ") + " " + v.logForm.desc.View(),
		fieldLabel(1, "Tags:        ") + " " + v.logForm.tags.View(),
		fieldLabel(2, "Minutes:     ") + " " + v.logForm.duration.View(),
		fieldLabel(3, "Date:        ") + " " + v.logForm.dateInput.View(),
		sMuted.Render("  Task ref:   " + v.logForm.taskID),
		"",
		sMuted.Render("tab next  ctrl+s save  esc cancel"),
	}, "\n")
}

func (v FutureView) renderSchedulePrompt(t model.Task, width int) string {
	return strings.Join([]string{
		sTitle.Render("Schedule to a day…"),
		sMuted.Render("→ " + t.Title),
		strings.Repeat("─", min(width, 36)),
		"",
		"  Date: " + v.scheduleInput.View(),
		"",
		sMuted.Render("  today, tomorrow, YYYY-MM-DD"),
		sMuted.Render("  enter to confirm  esc to cancel"),
	}, "\n")
}
