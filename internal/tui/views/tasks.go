package views

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/roramirez/pitlist/internal/model"
	"github.com/roramirez/pitlist/internal/storage"
)

type TasksMsg struct {
	Plan   *model.DayPlan
	ActLog *model.ActivityLog
}
type TasksSavedMsg struct{}

type detailMode int

const (
	detailNormal detailMode = iota
	detailEditNotes
	detailLogActivity
	detailCarry
	detailEditTask
)

const taskFormFields = 4 // title, context, labels, priority

type taskForm struct {
	focusIdx   int
	title      textinput.Model
	contextIdx int      // index into contexts slice; -1 = no context
	contexts   []string // options from config
	labels     textinput.Model
	priority   textinput.Model
	editID     string // empty = new task, non-empty = editing existing
}

func newTaskForm(editID, title, context string, contexts []string, labels, priority string) taskForm {
	ti := textinput.New()
	ti.Placeholder = "Task title"
	ti.CharLimit = 200
	ti.SetValue(title)
	ti.Focus()

	li := textinput.New()
	li.Placeholder = "labels space-separated (auth infra)"
	li.SetValue(labels)

	pi := textinput.New()
	pi.Placeholder = "low | medium | high"
	pi.CharLimit = 10
	if priority == "" {
		priority = "medium"
	}
	pi.SetValue(priority)

	// Find index of current context
	ctxIdx := -1
	for i, c := range contexts {
		if c == context {
			ctxIdx = i
			break
		}
	}

	return taskForm{title: ti, contextIdx: ctxIdx, contexts: contexts, labels: li, priority: pi, editID: editID}
}

// contextValue returns the selected context string, or "" if none.
func (f taskForm) contextValue() string {
	if f.contextIdx < 0 || f.contextIdx >= len(f.contexts) {
		return ""
	}
	return f.contexts[f.contextIdx]
}

// contextDisplay renders the selector for the context field.
func (f taskForm) contextDisplay(focused bool) string {
	accent := lipgloss.NewStyle().Foreground(lipgloss.Color("63"))
	muted := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	label := "—"
	if f.contextIdx >= 0 && f.contextIdx < len(f.contexts) {
		label = f.contexts[f.contextIdx]
	}

	if focused {
		return accent.Render("← ") + lipgloss.NewStyle().Bold(true).Render(label) + accent.Render(" →")
	}
	return muted.Render("  " + label)
}

const quickLogFields = 4 // desc, tags, duration, date

// quickLogForm is the inline activity form shown in the detail pane.
type quickLogForm struct {
	focusIdx  int
	desc      textinput.Model
	tags      textinput.Model
	duration  textinput.Model
	dateInput textinput.Model
	taskID    string // pre-filled, read-only
}

func newQuickLogForm(taskID string, defaultDate time.Time) quickLogForm {
	_ = defaultDate
	desc := textinput.New()
	desc.Placeholder = "What did you do?"
	desc.CharLimit = 200
	desc.Focus()

	tags := textinput.New()
	tags.Placeholder = "tags space-separated"

	dur := textinput.New()
	dur.Placeholder = "minutes (optional)"
	dur.CharLimit = 4

	di := textinput.New()
	di.CharLimit = 16
	di.SetValue(time.Now().Format("2006-01-02T15:04"))

	return quickLogForm{taskID: taskID, desc: desc, tags: tags, duration: dur, dateInput: di}
}

type LinkedActivitiesMsg struct{ Entries []model.ActivityEntry }
type GlobalTasksMsg struct{ Tasks []*model.Task }

type TasksView struct {
	store            *storage.YAMLStore
	date             time.Time
	plan             *model.DayPlan
	actLog           *model.ActivityLog
	linkedActivities []model.ActivityEntry
	globalResults    []*model.Task // non-nil when filter is active and global
	contexts         []string      // ordered list of context names from config
	cursor           int
	pane             int // 0=list, 1=detail
	weekMode         bool
	adding           bool
	input            textinput.Model
	detailMode       detailMode
	notesArea        textarea.Model
	logForm          quickLogForm
	tForm            taskForm
	carryInput       textinput.Model
	carryTaskID      string
	width            int
	height           int
	filter           TaskFilter
}

type TaskFilter struct {
	Labels   []string
	Statuses []model.TaskStatus
	Search   string
}

func NewTasksView(store *storage.YAMLStore, date time.Time, contexts ...string) TasksView {
	ti := textinput.New()
	ti.Placeholder = "Task title…"
	ti.CharLimit = 200

	ta := textarea.New()
	ta.Placeholder = "Task notes…"
	ta.ShowLineNumbers = false

	ci := textinput.New()
	ci.CharLimit = 10

	return TasksView{
		store:      store,
		date:       date,
		plan:       &model.DayPlan{Date: date, Tasks: []model.Task{}},
		input:      ti,
		notesArea:  ta,
		carryInput: ci,
		contexts:   contexts,
		filter:     TaskFilter{Statuses: []model.TaskStatus{model.StatusTodo, model.StatusInProgress}},
	}
}

func (v TasksView) Load() tea.Cmd {
	return func() tea.Msg {
		plan, err := v.store.GetDayPlan(v.date)
		if err != nil {
			return errMsg{err}
		}
		actLog, err := v.store.GetActivityLog(v.date)
		if err != nil {
			return errMsg{err}
		}
		return TasksMsg{Plan: plan, ActLog: actLog}
	}
}

type errMsg struct{ err error }

func (v TasksView) loadLinkedActivities(taskID string) tea.Cmd {
	if taskID == "" {
		return nil
	}
	return func() tea.Msg {
		task, date, err := v.store.GetTaskByID(taskID)
		if err != nil {
			return LinkedActivitiesMsg{}
		}
		entries, err := v.store.GetActivitiesByRefs(task.ActivityRefs, date)
		if err != nil {
			return errMsg{err}
		}
		result := make([]model.ActivityEntry, len(entries))
		for i, e := range entries {
			result[i] = *e
		}
		return LinkedActivitiesMsg{Entries: result}
	}
}

func (v TasksView) loadMsg() tea.Msg {
	plan, err := v.store.GetDayPlan(v.date)
	if err != nil {
		return errMsg{err}
	}
	actLog, err := v.store.GetActivityLog(v.date)
	if err != nil {
		return errMsg{err}
	}
	return TasksMsg{Plan: plan, ActLog: actLog}
}

func (v TasksView) Update(msg tea.Msg) (TasksView, tea.Cmd) {
	switch msg := msg.(type) {
	case TasksMsg:
		v.plan = msg.Plan
		v.actLog = msg.ActLog
		if v.cursor >= len(v.filteredTasks()) {
			v.cursor = max(0, len(v.filteredTasks())-1)
		}
		tasks := v.filteredTasks()
		if len(tasks) > 0 {
			return v, v.loadLinkedActivities(tasks[v.cursor].ID)
		}
		return v, nil

	case LinkedActivitiesMsg:
		v.linkedActivities = msg.Entries
		return v, nil

	case GlobalTasksMsg:
		v.globalResults = msg.Tasks
		v.cursor = 0
		return v, nil

	case tea.KeyMsg:
		if v.adding {
			return v.updateAdding(msg)
		}
		switch v.detailMode {
		case detailEditNotes:
			return v.updateNotes(msg)
		case detailLogActivity:
			return v.updateLogForm(msg)
		case detailCarry:
			return v.updateCarry(msg)
		case detailEditTask:
			return v.updateTaskForm(msg)
		}
		if v.adding {
			return v.updateAdding(msg)
		}
		return v.updateNormal(msg)

	case tea.WindowSizeMsg:
		v.width = msg.Width
		v.height = msg.Height
	}
	return v, nil
}

func (v TasksView) updateNormal(msg tea.KeyMsg) (TasksView, tea.Cmd) {
	tasks := v.filteredTasks()
	switch msg.String() {
	case "j", "down":
		if v.cursor < len(tasks)-1 {
			v.cursor++
			if len(tasks) > 0 {
				return v, v.loadLinkedActivities(tasks[v.cursor].ID)
			}
		}
	case "k", "up":
		if v.cursor > 0 {
			v.cursor--
			if len(tasks) > 0 {
				return v, v.loadLinkedActivities(tasks[v.cursor].ID)
			}
		}
	case "h", "left", "[":
		if v.pane == 0 {
			v.date = v.date.AddDate(0, 0, -1)
			v.cursor = 0
			return v, v.Load()
		}
	case "l", "right", "]":
		if v.pane == 0 {
			v.date = v.date.AddDate(0, 0, 1)
			v.cursor = 0
			return v, v.Load()
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
	case "c":
		if len(tasks) > 0 {
			tomorrow := v.date.AddDate(0, 0, 1).Format("2006-01-02")
			v.carryTaskID = tasks[v.cursor].ID
			v.carryInput.Reset()
			v.carryInput.SetValue(tomorrow)
			v.carryInput.Focus()
			v.detailMode = detailCarry
			v.pane = 1
			return v, textinput.Blink
		}
	case "D":
		if len(tasks) > 0 {
			return v, v.deleteTask(tasks[v.cursor].ID)
		}
	case "w":
		v.weekMode = !v.weekMode
	case "e":
		if len(tasks) > 0 {
			t := tasks[v.cursor]
			v.tForm = newTaskForm(t.ID, t.Title, t.Context, v.contexts, strings.Join(t.Labels, " "), string(t.Priority))
			v.detailMode = detailEditTask
			v.pane = 1
			return v, textinput.Blink
		}
	case "n":
		// Edit notes for selected task
		if len(tasks) > 0 {
			v.pane = 1
			v.detailMode = detailEditNotes
			v.notesArea.Reset()
			v.notesArea.SetValue(tasks[v.cursor].Notes)
			v.notesArea.Focus()
			return v, textarea.Blink
		}
	case "L":
		// Log activity linked to selected task
		if len(tasks) > 0 {
			v.pane = 1
			v.detailMode = detailLogActivity
			v.logForm = newQuickLogForm(tasks[v.cursor].ID, v.date)
			return v, textinput.Blink
		}
	}
	return v, nil
}

// updateNotes handles textarea input for editing task notes.
func (v TasksView) updateNotes(msg tea.KeyMsg) (TasksView, tea.Cmd) {
	switch msg.String() {
	case "esc":
		v.detailMode = detailNormal
		v.notesArea.Blur()
		return v, nil
	case "ctrl+s":
		tasks := v.filteredTasks()
		if len(tasks) == 0 {
			v.detailMode = detailNormal
			return v, nil
		}
		notes := v.notesArea.Value()
		id := tasks[v.cursor].ID
		v.detailMode = detailNormal
		v.notesArea.Blur()
		return v, v.saveNotes(id, notes)
	default:
		var cmd tea.Cmd
		v.notesArea, cmd = v.notesArea.Update(msg)
		return v, cmd
	}
}

func (v TasksView) saveNotes(id, notes string) tea.Cmd {
	return func() tea.Msg {
		plan, err := v.store.GetDayPlan(v.date)
		if err != nil {
			return errMsg{err}
		}
		for i := range plan.Tasks {
			if plan.Tasks[i].ID == id {
				plan.Tasks[i].Notes = notes
				plan.Tasks[i].UpdatedAt = time.Now().UTC()
				break
			}
		}
		if err := v.store.SaveDayPlan(plan); err != nil {
			return errMsg{err}
		}
		return v.loadMsg()
	}
}

// updateLogForm handles the quick activity log form inside the detail pane.
func (v TasksView) updateLogForm(msg tea.KeyMsg) (TasksView, tea.Cmd) {
	switch msg.String() {
	case "esc":
		v.detailMode = detailNormal
		return v, nil
	case "ctrl+s":
		cmd := v.submitLogForm()
		v.detailMode = detailNormal
		return v, cmd
	case "tab", "down":
		v.logForm.focusIdx = (v.logForm.focusIdx + 1) % quickLogFields
		return v.focusLogField()
	case "shift+tab", "up":
		v.logForm.focusIdx = (v.logForm.focusIdx + quickLogFields - 1) % quickLogFields
		return v.focusLogField()
	case "enter":
		if v.logForm.focusIdx == quickLogFields-1 {
			v.detailMode = detailNormal
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

func (v TasksView) updateCarry(msg tea.KeyMsg) (TasksView, tea.Cmd) {
	switch msg.String() {
	case "enter", "ctrl+s":
		raw := strings.TrimSpace(v.carryInput.Value())
		destDate, err := time.Parse("2006-01-02", raw)
		if err != nil {
			// invalid date — stay in prompt
			return v, nil
		}
		taskID := v.carryTaskID
		v.detailMode = detailNormal
		v.carryInput.Blur()
		return v, v.carryTaskTo(taskID, destDate)
	case "esc":
		v.detailMode = detailNormal
		v.carryInput.Blur()
	default:
		var cmd tea.Cmd
		v.carryInput, cmd = v.carryInput.Update(msg)
		return v, cmd
	}
	return v, nil
}

func (v TasksView) carryTaskTo(id string, destDate time.Time) tea.Cmd {
	return func() tea.Msg {
		task, srcDate, err := v.store.GetTaskByID(id)
		if err != nil {
			return errMsg{err}
		}
		now := time.Now().UTC()

		srcPlan, err := v.store.GetDayPlan(srcDate)
		if err != nil {
			return errMsg{err}
		}
		remaining := make([]model.Task, 0, len(srcPlan.Tasks))
		for _, t := range srcPlan.Tasks {
			if t.ID != id {
				remaining = append(remaining, t)
			}
		}
		srcPlan.Tasks = remaining
		if err := v.store.SaveDayPlan(srcPlan); err != nil {
			return errMsg{err}
		}

		destPlan, err := v.store.GetDayPlan(destDate)
		if err != nil {
			return errMsg{err}
		}
		moved := *task
		moved.UpdatedAt = now
		destPlan.Tasks = append(destPlan.Tasks, moved)
		if err := v.store.SaveDayPlan(destPlan); err != nil {
			return errMsg{err}
		}

		actLog, err := v.store.GetActivityLog(srcDate)
		if err != nil {
			return errMsg{err}
		}
		entry := model.ActivityEntry{
			ID:          storage.NextActivityID(actLog),
			Timestamp:   now,
			Description: "Carried to " + destDate.Format("2006-01-02") + ": " + task.Title,
			Tags:        []string{"carried"},
			TaskRef:     task.ID,
		}
		actLog.Entries = append(actLog.Entries, entry)
		if err := v.store.SaveActivityLog(actLog); err != nil {
			return errMsg{err}
		}
		_ = v.store.AddActivityRefToTask(id, model.ActivityRef{
			ID:   entry.ID,
			Date: srcDate.Format("2006-01-02"),
		})
		return v.loadMsg()
	}
}

func (v TasksView) focusLogField() (TasksView, tea.Cmd) {
	v.logForm.desc.Blur()
	v.logForm.tags.Blur()
	v.logForm.duration.Blur()
	v.logForm.dateInput.Blur()
	// When focusing the date field, recompute from duration
	if v.logForm.focusIdx == 3 {
		dur := 0
		if d, err := strconv.Atoi(strings.TrimSpace(v.logForm.duration.Value())); err == nil {
			dur = d
		}
		ts := time.Now().Add(-time.Duration(dur) * time.Minute)
		v.logForm.dateInput.SetValue(ts.Format("2006-01-02T15:04"))
	}
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

func (v TasksView) submitLogForm() tea.Cmd {
	desc := strings.TrimSpace(v.logForm.desc.Value())
	if desc == "" {
		v.detailMode = detailNormal
		return nil
	}
	var tags []string
	for _, t := range strings.Fields(v.logForm.tags.Value()) {
		tags = append(tags, t)
	}
	dur := 0
	if d, err := strconv.Atoi(strings.TrimSpace(v.logForm.duration.Value())); err == nil {
		dur = d
	}
	taskID := v.logForm.taskID

	// Parse the editable date field; fall back to now-duration
	now := time.Now()
	ts := now.Add(-time.Duration(dur) * time.Minute)
	if raw := strings.TrimSpace(v.logForm.dateInput.Value()); raw != "" {
		if t, err := time.ParseInLocation("2006-01-02T15:04", raw, time.Local); err == nil {
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
			_ = v.store.AddActivityRefToTask(taskID, model.ActivityRef{
				ID:   entry.ID,
				Date: entryDate.Format("2006-01-02"),
			})
		}
		return v.loadMsg()
	}
}

func (v TasksView) updateAdding(msg tea.KeyMsg) (TasksView, tea.Cmd) {
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

func (v TasksView) updateTaskForm(msg tea.KeyMsg) (TasksView, tea.Cmd) {
	switch msg.String() {
	case "tab", "down":
		v.tForm.focusIdx = (v.tForm.focusIdx + 1) % taskFormFields
		return v.focusTaskField()
	case "shift+tab", "up":
		v.tForm.focusIdx = (v.tForm.focusIdx + taskFormFields - 1) % taskFormFields
		return v.focusTaskField()
	case "ctrl+s":
		v.detailMode = detailNormal
		return v, v.saveEditTask()
	case "enter":
		if v.tForm.focusIdx == taskFormFields-1 {
			v.detailMode = detailNormal
			return v, v.saveEditTask()
		}
		v.tForm.focusIdx = (v.tForm.focusIdx + 1) % taskFormFields
		return v.focusTaskField()
	case "esc":
		v.detailMode = detailNormal
	default:
		v = v.handleFormKey(msg)
		return v, nil
	}
	return v, nil
}

// handleFormKey routes key events to the right field, handling context cycling.
func (v TasksView) handleFormKey(msg tea.KeyMsg) TasksView {
	n := len(v.tForm.contexts)
	switch v.tForm.focusIdx {
	case 0:
		v.tForm.title, _ = v.tForm.title.Update(msg)
	case 1: // context selector
		switch msg.String() {
		case "left", "h":
			if n > 0 {
				v.tForm.contextIdx = (v.tForm.contextIdx - 1 + n + 1) % (n + 1) - 1
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

func (v TasksView) focusTaskField() (TasksView, tea.Cmd) {
	v.tForm.title.Blur()
	v.tForm.labels.Blur()
	v.tForm.priority.Blur()
	switch v.tForm.focusIdx {
	case 0:
		v.tForm.title.Focus()
		return v, textinput.Blink
	case 1:
		// context is a selector, no blink needed
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

func (v TasksView) saveNewTask() tea.Cmd {
	title := strings.TrimSpace(v.tForm.title.Value())
	context := v.tForm.contextValue()
	labels := parseLabels(v.tForm.labels.Value())
	priority := parsePriority(v.tForm.priority.Value())
	return func() tea.Msg {
		plan, err := v.store.GetDayPlan(v.date)
		if err != nil {
			return errMsg{err}
		}
		now := time.Now().UTC()
		task := model.Task{
			ID:        storage.NextTaskID(plan),
			Title:     title,
			Context:   context,
			Labels:    labels,
			Status:    model.StatusTodo,
			Priority:  priority,
			CreatedAt: now,
			UpdatedAt: now,
		}
		plan.Tasks = append(plan.Tasks, task)
		if err := v.store.SaveDayPlan(plan); err != nil {
			return errMsg{err}
		}
		return v.loadMsg()
	}
}

func (v TasksView) saveEditTask() tea.Cmd {
	id := v.tForm.editID
	title := strings.TrimSpace(v.tForm.title.Value())
	context := v.tForm.contextValue()
	labels := parseLabels(v.tForm.labels.Value())
	priority := parsePriority(v.tForm.priority.Value())
	return func() tea.Msg {
		plan, err := v.store.GetDayPlan(v.date)
		if err != nil {
			return errMsg{err}
		}
		for i := range plan.Tasks {
			if plan.Tasks[i].ID == id {
				if title != "" {
					plan.Tasks[i].Title = title
				}
				plan.Tasks[i].Context = context
				plan.Tasks[i].Labels = labels
				plan.Tasks[i].Priority = priority
				plan.Tasks[i].UpdatedAt = time.Now().UTC()
				break
			}
		}
		if err := v.store.SaveDayPlan(plan); err != nil {
			return errMsg{err}
		}
		return v.loadMsg()
	}
}

func parseLabels(raw string) []string {
	var out []string
	for _, l := range strings.Fields(raw) {
		if l != "" {
			out = append(out, l)
		}
	}
	return out
}

func parsePriority(raw string) model.Priority {
	switch strings.TrimSpace(strings.ToLower(raw)) {
	case "low", "l":
		return model.PriorityLow
	case "high", "h":
		return model.PriorityHigh
	default:
		return model.PriorityMedium
	}
}

func (v TasksView) toggleDone(id string) tea.Cmd {
	return func() tea.Msg {
		plan, err := v.store.GetDayPlan(v.date)
		if err != nil {
			return errMsg{err}
		}
		now := time.Now().UTC()
		for i := range plan.Tasks {
			if plan.Tasks[i].ID == id {
				if plan.Tasks[i].Status == model.StatusDone {
					plan.Tasks[i].Status = model.StatusTodo
					plan.Tasks[i].DoneAt = nil
				} else {
					plan.Tasks[i].Status = model.StatusDone
					plan.Tasks[i].DoneAt = &now
				}
				plan.Tasks[i].UpdatedAt = now
				break
			}
		}
		if err := v.store.SaveDayPlan(plan); err != nil {
			return errMsg{err}
		}
		return v.loadMsg()
	}
}

func (v TasksView) deleteTask(id string) tea.Cmd {
	return func() tea.Msg {
		plan, err := v.store.GetDayPlan(v.date)
		if err != nil {
			return errMsg{err}
		}
		remaining := make([]model.Task, 0, len(plan.Tasks))
		for _, t := range plan.Tasks {
			if t.ID != id {
				remaining = append(remaining, t)
			}
		}
		plan.Tasks = remaining
		if err := v.store.SaveDayPlan(plan); err != nil {
			return errMsg{err}
		}
		return v.loadMsg()
	}
}

func (v TasksView) filteredTasks() []model.Task {
	var raw []model.Task
	if v.globalResults != nil {
		for _, t := range v.globalResults {
			raw = append(raw, *t)
		}
	} else if v.plan != nil {
		for _, t := range v.plan.Tasks {
			if matchStatus(t.Status, v.filter.Statuses) {
				raw = append(raw, t)
			}
		}
	}
	if len(v.contexts) == 0 {
		return raw
	}
	// Return tasks ordered by context so cursor index matches visual order
	return sortByContext(raw, v.contexts)
}

// sortByContext returns tasks grouped in context order: configured contexts
// first (in order), then no-context tasks, then carried tasks last.
func sortByContext(tasks []model.Task, contexts []string) []model.Task {
	rank := make(map[string]int, len(contexts))
	for i, c := range contexts {
		rank[c] = i
	}

	var groups [][]model.Task
	byCtx := make(map[string][]model.Task)
	var noCtx, carried []model.Task

	for _, t := range tasks {
		if t.CarryFrom != "" {
			carried = append(carried, t)
			continue
		}
		byCtx[t.Context] = append(byCtx[t.Context], t)
	}

	for _, c := range contexts {
		if ts := byCtx[c]; len(ts) > 0 {
			groups = append(groups, ts)
			delete(byCtx, c)
		}
	}
	// remaining contexts not in config
	for _, ts := range byCtx {
		noCtx = append(noCtx, ts...)
	}
	if len(noCtx) > 0 {
		groups = append(groups, noCtx)
	}
	if len(carried) > 0 {
		groups = append(groups, carried)
	}

	var out []model.Task
	for _, g := range groups {
		out = append(out, g...)
	}
	_ = rank
	return out
}

func matchStatus(s model.TaskStatus, statuses []model.TaskStatus) bool {
	if len(statuses) == 0 {
		return true
	}
	for _, st := range statuses {
		if s == st {
			return true
		}
	}
	return false
}

func (v TasksView) View(width, height int) string {
	listWidth := width / 2
	detailWidth := width - listWidth - 3

	listContent := v.renderList(listWidth - 4)
	detailContent := v.renderDetail(detailWidth - 4)

	listStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("238")).
		Width(listWidth).
		Height(height - 2).
		Padding(0, 1)

	detailStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("238")).
		Width(detailWidth).
		Height(height - 2).
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

func (v TasksView) renderList(width int) string {
	header := v.renderDayNav()
	var lines []string
	lines = append(lines, header, "")

	if v.adding {
		lines = append(lines, v.renderTaskFormInline()...)
	}

	tasks := v.filteredTasks()

	if len(tasks) == 0 && !v.adding {
		lines = append(lines, lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("  No tasks. Press 'a' to add one."))
	} else {
		lines = append(lines, v.renderTasksByContext(tasks, width)...)
	}

	return strings.Join(lines, "\n")
}

// renderTasksByContext renders the already-sorted tasks list with context
// section headers when the context changes. Tasks arrive pre-sorted from
// filteredTasks() so cursor index is always consistent with visual position.
func (v TasksView) renderTasksByContext(tasks []model.Task, width int) []string {
	muted := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	bold := lipgloss.NewStyle().Bold(true)

	useHeaders := len(v.contexts) > 0 && hasMultipleContexts(tasks)
	var lines []string
	prevCtx := "\x00" // sentinel to force first header

	for i, t := range tasks {
		selected := i == v.cursor

		// Section header when context changes (skip for carried tasks)
		if useHeaders && t.CarryFrom == "" && t.Context != prevCtx {
			label := t.Context
			if label == "" {
				label = "—"
			}
			sep := strings.Repeat("─", max(0, width-len(label)-4))
			lines = append(lines, "", bold.Render("  "+label)+"  "+muted.Render(sep))
			prevCtx = t.Context
		}
		if useHeaders && t.CarryFrom != "" && prevCtx != "carried" {
			lines = append(lines, "", muted.Render("  ── carried ──"))
			prevCtx = "carried"
		}

		lines = append(lines, renderTaskLine(t, selected, width))
	}
	return lines
}

func hasMultipleContexts(tasks []model.Task) bool {
	seen := map[string]bool{}
	for _, t := range tasks {
		if t.CarryFrom != "" {
			continue
		}
		seen[t.Context] = true
		if len(seen) > 1 {
			return true
		}
	}
	return false
}

func (v TasksView) renderDayNav() string {
	dateStr := v.date.Format("Mon Jan 02")
	muted := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	bold := lipgloss.NewStyle().Bold(true)
	return fmt.Sprintf("%s %s %s", muted.Render("←"), bold.Render(dateStr), muted.Render("→"))
}

func renderTaskLine(t model.Task, selected bool, width int) string {
	check := "[ ]"
	titleStyle := lipgloss.NewStyle()

	switch t.Status {
	case model.StatusDone:
		check = "[x]"
		titleStyle = titleStyle.Foreground(lipgloss.Color("240")).Strikethrough(true)
	case model.StatusInProgress:
		check = "[~]"
	case model.StatusCancelled:
		check = "[-]"
		titleStyle = titleStyle.Foreground(lipgloss.Color("240"))
	}

	priority := ""
	if t.Priority == model.PriorityHigh {
		priority = lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render(" !")
	}

	carryMark := ""
	if t.CarryFrom != "" {
		carryMark = lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render(" ↑")
	}

	hasNotes := ""
	if t.Notes != "" {
		hasNotes = lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render(" ¶")
	}

	line := fmt.Sprintf("  %s %s%s%s%s", check, titleStyle.Render(t.Title), priority, carryMark, hasNotes)
	if selected {
		line = lipgloss.NewStyle().Background(lipgloss.Color("236")).Render(line)
	}
	return line
}

func (v TasksView) renderDetail(width int) string {
	tasks := v.filteredTasks()
	if len(tasks) == 0 {
		return lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Render("No task selected.")
	}
	if v.cursor >= len(tasks) {
		return ""
	}
	t := tasks[v.cursor]

	switch v.detailMode {
	case detailEditNotes:
		return v.renderNotesEditor(t, width)
	case detailLogActivity:
		return v.renderLogForm(t, width)
	case detailCarry:
		return v.renderCarryPrompt(t, width)
	case detailEditTask:
		return v.renderTaskFormDetail(t, width)
	default:
		return v.renderTaskDetail(t, width)
	}
}

func (v TasksView) renderTaskDetail(t model.Task, width int) string {
	bold := lipgloss.NewStyle().Bold(true)
	muted := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	var lines []string
	lines = append(lines, bold.Render(t.Title))
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
	if t.DueDate != "" {
		lines = append(lines, fmt.Sprintf("Due:      %s", t.DueDate))
	}

	if t.Notes != "" {
		lines = append(lines, "", bold.Render("Notes:"), t.Notes)
	}

	linked := v.linkedActivities
	if len(linked) > 0 {
		totalMin := 0
		for _, e := range linked {
			totalMin += e.DurationMin
		}

		header := bold.Render("Activity:")
		if totalMin > 0 {
			total := fmt.Sprintf("%dh %02dm", totalMin/60, totalMin%60)
			header += "  " + lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("∑ "+total)
		}
		lines = append(lines, "", header)

		for _, e := range linked {
			dur := ""
			if e.DurationMin > 0 {
				dur = fmt.Sprintf(" %dm", e.DurationMin)
			}
			tags := ""
			if len(e.Tags) > 0 {
				tags = muted.Render(" [" + strings.Join(e.Tags, ", ") + "]")
			}
			lines = append(lines, fmt.Sprintf("  %s%s  %s%s",
				muted.Render(e.Timestamp.Local().Format("Jan 02 15:04")),
				lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render(dur),
				e.Description,
				tags,
			))
		}
	}

	lines = append(lines, "", muted.Render("ID: "+t.ID))
	lines = append(lines, "", muted.Render("n notes  L log activity  d done  c carry  tab ←list"))

	return strings.Join(lines, "\n")
}

func (v TasksView) renderTaskFormInline() []string {
	bold := lipgloss.NewStyle().Bold(true)
	muted := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	fl := func(idx int, label string) string {
		if v.tForm.focusIdx == idx {
			return bold.Render("> " + label)
		}
		return muted.Render("  " + label)
	}
	return []string{
		bold.Render("── New task ──"),
		fl(0, "Title:    ") + " " + v.tForm.title.View(),
		fl(1, "Context:  ") + " " + v.tForm.contextDisplay(v.tForm.focusIdx == 1),
		fl(2, "Labels:   ") + " " + v.tForm.labels.View(),
		fl(3, "Priority: ") + " " + v.tForm.priority.View(),
		muted.Render("  tab next  ←/→ context  ctrl+s save  esc cancel"),
		"",
	}
}

func (v TasksView) renderTaskFormDetail(t model.Task, width int) string {
	bold := lipgloss.NewStyle().Bold(true)
	muted := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	fl := func(idx int, label string) string {
		if v.tForm.focusIdx == idx {
			return bold.Render("> " + label)
		}
		return muted.Render("  " + label)
	}
	return strings.Join([]string{
		bold.Render("Edit task"),
		muted.Render("ID: " + t.ID),
		strings.Repeat("─", min(width, 36)),
		"",
		fl(0, "Title:    ") + " " + v.tForm.title.View(),
		fl(1, "Context:  ") + " " + v.tForm.contextDisplay(v.tForm.focusIdx == 1),
		fl(2, "Labels:   ") + " " + v.tForm.labels.View(),
		fl(3, "Priority: ") + " " + v.tForm.priority.View(),
		"",
		muted.Render("  tab next  ←/→ context  ctrl+s save  esc cancel"),
	}, "\n")
}

func (v TasksView) renderCarryPrompt(t model.Task, width int) string {
	bold := lipgloss.NewStyle().Bold(true)
	muted := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))
	warn := lipgloss.NewStyle().Foreground(lipgloss.Color("214"))

	raw := strings.TrimSpace(v.carryInput.Value())
	hint := ""
	if _, err := time.Parse("2006-01-02", raw); err != nil && raw != "" {
		hint = warn.Render("  invalid date")
	}

	return strings.Join([]string{
		bold.Render("Carry task to…"),
		muted.Render("→ " + t.Title),
		strings.Repeat("─", min(width, 36)),
		"",
		"  Date: " + v.carryInput.View() + hint,
		"",
		muted.Render("  enter to confirm  esc to cancel"),
	}, "\n")
}

func (v TasksView) renderNotesEditor(t model.Task, width int) string {
	bold := lipgloss.NewStyle().Bold(true)
	muted := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	v.notesArea.SetWidth(width - 2)
	v.notesArea.SetHeight(10)

	return strings.Join([]string{
		bold.Render("Notes: " + t.Title),
		strings.Repeat("─", min(len(t.Title)+8, width)),
		"",
		v.notesArea.View(),
		"",
		muted.Render("ctrl+s save  esc cancel"),
	}, "\n")
}

func (v TasksView) renderLogForm(t model.Task, width int) string {
	bold := lipgloss.NewStyle().Bold(true)
	muted := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	fieldLabel := func(idx int, label string) string {
		if v.logForm.focusIdx == idx {
			return bold.Render("> " + label)
		}
		return muted.Render("  " + label)
	}

	return strings.Join([]string{
		bold.Render("Log activity"),
		muted.Render("→ " + t.Title),
		strings.Repeat("─", min(width, 36)),
		"",
		fieldLabel(0, "Description: ") + " " + v.logForm.desc.View(),
		fieldLabel(1, "Tags:        ") + " " + v.logForm.tags.View(),
		fieldLabel(2, "Minutes:     ") + " " + v.logForm.duration.View(),
		fieldLabel(3, "Date:        ") + " " + v.logForm.dateInput.View(),
		muted.Render("  Task ref:   " + v.logForm.taskID),
		"",
		muted.Render("tab next  ctrl+s save  esc cancel"),
	}, "\n")
}

func (v TasksView) Date() time.Time      { return v.date }
func (v TasksView) Contexts() []string   { return v.contexts }

func (v TasksView) SetFilter(f TaskFilter) (TasksView, tea.Cmd) {
	v.filter = f
	isGlobal := len(f.Labels) > 0 || f.Search != ""
	if !isGlobal {
		v.globalResults = nil
		return v, v.Load()
	}
	return v, func() tea.Msg {
		sf := storage.TaskFilter{
			Labels:   f.Labels,
			Search:   f.Search,
			Statuses: f.Statuses,
		}
		tasks, err := v.store.ListTasks(sf)
		if err != nil {
			return errMsg{err}
		}
		return GlobalTasksMsg{Tasks: tasks}
	}
}

func (v TasksView) IsInputActive() bool {
	return v.adding || v.detailMode != detailNormal
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
