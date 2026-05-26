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

const (
	scheduleToday    = "today"
	scheduleTomorrow = "tomorrow"
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
		v.pane = 1 - v.pane
	default:
		return v.handleFutureAction(msg, tasks)
	}
	return v, nil
}

func (v FutureView) handleFutureAction(msg tea.KeyMsg, tasks []model.Task) (FutureView, tea.Cmd) {
	if msg.String() == "a" {
		if v.pane == 0 {
			v.tForm = newTaskForm("", "", "", v.contexts, "", "medium")
			v.adding = true
			return v, textinput.Blink
		}
		return v, nil
	}
	if len(tasks) == 0 {
		return v, nil
	}
	return v.handleExistingFutureAction(msg, tasks[v.cursor])
}

func (v FutureView) handleExistingFutureAction(msg tea.KeyMsg, t model.Task) (FutureView, tea.Cmd) {
	switch msg.String() {
	case "d":
		return v, v.toggleDone(t.ID)
	case "D":
		return v, v.deleteTask(t.ID)
	case "e":
		v.tForm = newTaskForm(t.ID, t.Title, t.Context, v.contexts, strings.Join(t.Labels, " "), string(t.Priority))
		v.detailMode = futureDetailEditTask
		v.pane = 1
		return v, textinput.Blink
	case "n":
		v.pane = 1
		v.detailMode = futureDetailEditNotes
		v.notesArea.Reset()
		v.notesArea.SetValue(t.Notes)
		v.notesArea.Focus()
		return v, textarea.Blink
	case "L":
		v.pane = 1
		v.detailMode = futureDetailLogActivity
		v.logForm = newQuickLogForm(t.ID)
		return v, textinput.Blink
	case "s":
		return v.openSchedulePrompt(t.ID)
	}
	return v, nil
}

func (v FutureView) openSchedulePrompt(taskID string) (FutureView, tea.Cmd) {
	v.scheduleTaskID = taskID
	v.scheduleInput.Reset()
	v.scheduleInput.SetValue(scheduleToday)
	v.scheduleInput.Focus()
	v.detailMode = futureDetailSchedule
	v.pane = 1
	return v, textinput.Blink
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
	v.tForm = handleTaskFormKey(v.tForm, msg)
	return v
}

func (v FutureView) focusTaskField() (FutureView, tea.Cmd) {
	var blink bool
	v.tForm, blink = applyTaskFormFocus(v.tForm)
	if blink {
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
		v.logForm, cmd = updateLogFormField(v.logForm, msg)
		return v, cmd
	}
}

func (v FutureView) focusLogField() (FutureView, tea.Cmd) {
	v.logForm = blurAllLogFields(v.logForm)
	v.logForm = focusActiveLogField(v.logForm)
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
		return v.saveFutureListMsg(list)
	}
}

func (v FutureView) saveFutureListMsg(list *model.FutureList) tea.Msg {
	if err := v.store.SaveFutureList(list); err != nil {
		return errMsg{err}
	}
	return v.loadMsg()
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
		return v.saveFutureListMsg(list)
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
		return v.saveFutureListMsg(list)
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
		return v.saveFutureListMsg(list)
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
		return v.saveFutureListMsg(list)
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

// parseScheduleDate converts a keyword or YYYY-MM-DD string to a UTC day.
// Empty string and "today" both resolve to today; unknown strings return today.
func parseScheduleDate(raw string) time.Time {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	switch strings.ToLower(raw) {
	case "", scheduleToday:
		return today
	case scheduleTomorrow:
		return today.AddDate(0, 0, 1)
	default:
		if parsed, err := time.Parse(model.DateFormat, raw); err == nil {
			return parsed
		}
		return today
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

		day := parseScheduleDate(rawDate)
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

	listStyle := sPaneInactive.Width(listWidth).Height(height - 2)
	detailStyle := sPaneInactive.Width(detailWidth).Height(height - 2)

	if v.pane == 0 {
		listStyle = sPaneActive.Width(listWidth).Height(height - 2)
	} else {
		detailStyle = sPaneActive.Width(detailWidth).Height(height - 2)
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
	lines := renderTaskHeader(t, width)
	if t.Notes != "" {
		lines = append(lines, "", sTitle.Render("Notes:"), t.Notes)
	}
	lines = append(lines, renderLinkedActivities(v.linkedActivities)...)
	lines = append(lines, "", sMuted.Render("ID: "+t.ID))
	lines = append(lines, "", sMuted.Render("n notes  L log  d done  s schedule  e edit  D delete  tab ←list"))
	return strings.Join(lines, "\n")
}

func (v FutureView) renderTaskFormDetail(t model.Task, width int) string {
	return renderTaskEditFormShared(v.tForm, t.ID, width)
}

func (v FutureView) renderNotesEditor(t model.Task, width int) string {
	return renderNotesEditorShared(v.notesArea, t.Title, width)
}

func (v FutureView) renderLogForm(t model.Task, width int) string {
	return renderLogFormShared(v.logForm, t.Title, width)
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
