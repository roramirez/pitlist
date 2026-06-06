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
	detailActions
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
	label := "—"
	if f.contextIdx >= 0 && f.contextIdx < len(f.contexts) {
		label = f.contexts[f.contextIdx]
	}

	if focused {
		return sAccent.Render("← ") + sTitle.Render(label) + sAccent.Render(" →")
	}
	return sMuted.Render("  " + label)
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

func newQuickLogForm(taskID string) quickLogForm {
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
	di.SetValue(time.Now().Format(model.DateTimeFormat))

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
	actionCursor     int
	actionInput      textinput.Model
	actionAdding     bool
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

	ai := textinput.New()
	ai.Placeholder = "Action title…"
	ai.CharLimit = 200

	return TasksView{
		store:       store,
		date:        date,
		plan:        &model.DayPlan{Date: date, Tasks: []model.Task{}},
		input:       ti,
		notesArea:   ta,
		carryInput:  ci,
		actionInput: ai,
		contexts:    contexts,
		filter:      TaskFilter{Statuses: []model.TaskStatus{model.StatusTodo, model.StatusInProgress}},
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
		tasks := v.filteredTasks()
		if v.cursor >= len(tasks) {
			v.cursor = max(0, len(tasks)-1)
		}
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
		case detailActions:
			return v.updateActions(msg)
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
		return v.moveCursor(tasks, +1)
	case "k", "up":
		return v.moveCursor(tasks, -1)
	case "h", "left", "[":
		return v.navigateDay(-1)
	case "l", "right", "]":
		return v.navigateDay(+1)
	case "tab":
		v.pane = 1 - v.pane
	case "w":
		v.weekMode = !v.weekMode
	case "f":
		return v.toggleShowDone()
	default:
		return v.handleTaskAction(msg, tasks)
	}
	return v, nil
}

func (v TasksView) navigateDay(delta int) (TasksView, tea.Cmd) {
	if v.pane != 0 {
		return v, nil
	}
	v.date = v.date.AddDate(0, 0, delta)
	v.cursor = 0
	return v, v.Load()
}

func (v TasksView) handleTaskAction(msg tea.KeyMsg, tasks []model.Task) (TasksView, tea.Cmd) {
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
	return v.handleExistingTaskAction(msg, tasks[v.cursor])
}

func (v TasksView) handleExistingTaskAction(msg tea.KeyMsg, t model.Task) (TasksView, tea.Cmd) {
	switch msg.String() {
	case "d":
		return v, v.toggleDone(t.ID)
	case "c":
		return v.openCarryPrompt(t.ID)
	case "D":
		return v, v.deleteTask(t.ID)
	case "e":
		v.tForm = newTaskForm(t.ID, t.Title, t.Context, v.contexts, strings.Join(t.Labels, " "), string(t.Priority))
		v.detailMode = detailEditTask
		v.pane = 1
		return v, textinput.Blink
	case "n":
		return v.openNotesEditor(t)
	case "L":
		v.pane = 1
		v.detailMode = detailLogActivity
		v.logForm = newQuickLogForm(t.ID)
		return v, textinput.Blink
	case "A":
		v.pane = 1
		v.detailMode = detailActions
		v.actionCursor = 0
		v.actionAdding = false
		return v, nil
	}
	return v, nil
}

func (v TasksView) openCarryPrompt(taskID string) (TasksView, tea.Cmd) {
	tomorrow := v.date.AddDate(0, 0, 1).Format(model.DateFormat)
	v.carryTaskID = taskID
	v.carryInput.Reset()
	v.carryInput.SetValue(tomorrow)
	v.carryInput.Focus()
	v.detailMode = detailCarry
	v.pane = 1
	return v, textinput.Blink
}

func (v TasksView) openNotesEditor(t model.Task) (TasksView, tea.Cmd) {
	v.pane = 1
	v.detailMode = detailEditNotes
	v.notesArea.Reset()
	v.notesArea.SetValue(t.Notes)
	v.notesArea.Focus()
	return v, textarea.Blink
}

func (v TasksView) moveCursor(tasks []model.Task, delta int) (TasksView, tea.Cmd) {
	next := v.cursor + delta
	if next < 0 || next >= len(tasks) {
		return v, nil
	}
	v.cursor = next
	return v, v.loadLinkedActivities(tasks[next].ID)
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
		return v.savePlanMsg(plan)
	}
}

func (v TasksView) savePlanMsg(plan *model.DayPlan) tea.Msg {
	if err := v.store.SaveDayPlan(plan); err != nil {
		return errMsg{err}
	}
	return v.loadMsg()
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
		v.logForm, cmd = updateLogFormField(v.logForm, msg)
		return v, cmd
	}
}

func (v TasksView) updateCarry(msg tea.KeyMsg) (TasksView, tea.Cmd) {
	switch msg.String() {
	case "enter", "ctrl+s":
		raw := strings.TrimSpace(v.carryInput.Value())
		destDate, err := time.Parse(model.DateFormat, raw)
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
			Description: "Carried to " + destDate.Format(model.DateFormat) + ": " + task.Title,
			Tags:        []string{"carried"},
			TaskRef:     task.ID,
		}
		actLog.Entries = append(actLog.Entries, entry)
		if err := v.store.SaveActivityLog(actLog); err != nil {
			return errMsg{err}
		}
		_ = v.store.AddActivityRefToTask(id, model.ActivityRef{
			ID:   entry.ID,
			Date: srcDate.Format(model.DateFormat),
		})
		return v.loadMsg()
	}
}

func (v TasksView) updateActions(msg tea.KeyMsg) (TasksView, tea.Cmd) {
	tasks := v.filteredTasks()
	if len(tasks) == 0 || v.cursor >= len(tasks) {
		return v, nil
	}
	t := tasks[v.cursor]
	res := handleActionEditorKey(v.actionCursor, v.actionAdding, v.actionInput, t.Actions, msg)
	v.actionCursor, v.actionAdding, v.actionInput = res.cursor, res.adding, res.input
	if res.exitMode {
		v.detailMode = detailNormal
		return v, nil
	}
	return v, v.actionCmd(t.ID, res)
}

func (v TasksView) actionCmd(taskID string, res actionEditorResult) tea.Cmd {
	switch {
	case res.blink:
		return textinput.Blink
	case res.toggleID != "":
		return v.toggleAction(taskID, res.toggleID)
	case res.deleteID != "":
		return v.deleteAction(taskID, res.deleteID)
	case res.newTitle != "":
		return v.saveNewAction(taskID, res.newTitle)
	}
	return nil
}

func (v TasksView) saveNewAction(taskID, title string) tea.Cmd {
	return func() tea.Msg {
		plan, err := v.store.GetDayPlan(v.date)
		if err != nil {
			return errMsg{err}
		}
		plan.Tasks = applyActionAdd(plan.Tasks, taskID, title)
		return v.savePlanMsg(plan)
	}
}

func (v TasksView) toggleAction(taskID, actionID string) tea.Cmd {
	return func() tea.Msg {
		plan, err := v.store.GetDayPlan(v.date)
		if err != nil {
			return errMsg{err}
		}
		plan.Tasks = applyActionToggle(plan.Tasks, taskID, actionID)
		return v.savePlanMsg(plan)
	}
}

func (v TasksView) deleteAction(taskID, actionID string) tea.Cmd {
	return func() tea.Msg {
		plan, err := v.store.GetDayPlan(v.date)
		if err != nil {
			return errMsg{err}
		}
		plan.Tasks = applyActionDelete(plan.Tasks, taskID, actionID)
		return v.savePlanMsg(plan)
	}
}

func (v TasksView) focusLogField() (TasksView, tea.Cmd) {
	v.logForm = blurAllLogFields(v.logForm)
	// When focusing the date field, recompute from duration
	if v.logForm.focusIdx == 3 {
		dur := 0
		if d, err := strconv.Atoi(strings.TrimSpace(v.logForm.duration.Value())); err == nil {
			dur = d
		}
		ts := time.Now().Add(-time.Duration(dur) * time.Minute)
		v.logForm.dateInput.SetValue(ts.Format(model.DateTimeFormat))
	}
	v.logForm = focusActiveLogField(v.logForm)
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
			_ = v.store.AddActivityRefToTask(taskID, model.ActivityRef{
				ID:   entry.ID,
				Date: entryDate.Format(model.DateFormat),
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
	v.tForm = handleTaskFormKey(v.tForm, msg)
	return v
}

func (v TasksView) focusTaskField() (TasksView, tea.Cmd) {
	var blink bool
	v.tForm, blink = applyTaskFormFocus(v.tForm)
	if blink {
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
		return v.savePlanMsg(plan)
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
		return v.savePlanMsg(plan)
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
		return v.savePlanMsg(plan)
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
		return v.savePlanMsg(plan)
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

func (v TasksView) renderList(width int) string {
	header := v.renderDayNav()
	var lines []string
	lines = append(lines, header, "")

	if v.adding {
		lines = append(lines, v.renderTaskFormInline()...)
	}

	tasks := v.filteredTasks()

	if len(tasks) == 0 && !v.adding {
		lines = append(lines, sMuted.Render("  No tasks. Press 'a' to add one."))
	} else {
		lines = append(lines, v.renderTasksByContext(tasks, width)...)
	}

	return strings.Join(lines, "\n")
}

// renderTasksByContext renders the already-sorted tasks list with context
// section headers when the context changes. Tasks arrive pre-sorted from
// filteredTasks() so cursor index is always consistent with visual position.
func (v TasksView) renderTasksByContext(tasks []model.Task, width int) []string {
	useHeaders := len(v.contexts) > 0 && hasMultipleContexts(tasks)
	var lines []string
	prevCtx := sentinelCtx

	for i, t := range tasks {
		if useHeaders && t.CarryFrom == "" && t.Context != prevCtx {
			lines = append(lines, "", contextSectionHeader(t.Context, width))
			prevCtx = t.Context
		}
		if useHeaders && t.CarryFrom != "" && prevCtx != carriedCtx {
			lines = append(lines, "", sMuted.Render("  ── carried ──"))
			prevCtx = carriedCtx
		}
		lines = append(lines, renderTaskLine(t, i == v.cursor, width))
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
	nav := fmt.Sprintf("%s %s %s", sMuted.Render("←"), sTitle.Render(dateStr), sMuted.Render("→"))
	if v.showingDone() {
		nav += "  " + sAccent.Render("[+done]")
	}
	return nav
}

func renderTaskLine(t model.Task, selected bool, width int) string {
	check := "[ ]"
	var titleStyle lipgloss.Style

	switch t.Status {
	case model.StatusDone:
		check = "[x]"
		titleStyle = sDone
	case model.StatusInProgress:
		check = "[~]"
		titleStyle = lipgloss.NewStyle()
	case model.StatusCancelled:
		check = "[-]"
		titleStyle = sMuted
	default:
		titleStyle = lipgloss.NewStyle()
	}

	priority := ""
	if t.Priority == model.PriorityHigh {
		priority = sHigh.Render(" !")
	}

	carryMark := ""
	if t.CarryFrom != "" {
		carryMark = sCarried.Render(" ↑")
	}

	hasNotes := ""
	if t.Notes != "" {
		hasNotes = sMuted.Render(" ¶")
	}

	actBadge := ""
	if len(t.Actions) > 0 {
		actBadge = sMuted.Render(actionBadge(t.Actions))
	}

	line := fmt.Sprintf("  %s %s%s%s%s%s", check, titleStyle.Render(t.Title), priority, carryMark, hasNotes, actBadge)
	if selected {
		line = sSelected.Render(line)
	}
	return line
}

func (v TasksView) renderDetail(width int) string {
	tasks := v.filteredTasks()
	if len(tasks) == 0 {
		return sMuted.Render("No task selected.")
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
	case detailActions:
		return v.renderActionsEditor(t, width)
	default:
		return v.renderTaskDetail(t, width)
	}
}

func (v TasksView) renderTaskDetail(t model.Task, width int) string {
	lines := renderTaskHeader(t, width)
	if t.DueDate != "" {
		lines = append(lines, fmt.Sprintf("Due:      %s", t.DueDate))
	}
	if t.Notes != "" {
		lines = append(lines, "", sTitle.Render("Notes:"), t.Notes)
	}
	lines = append(lines, renderActionsDetailSection(t.Actions)...)
	lines = append(lines, renderLinkedActivities(v.linkedActivities)...)
	lines = append(lines, "", sMuted.Render("ID: "+t.ID))
	lines = append(lines, "", sMuted.Render("n notes  L log  d done  c carry  A actions  tab ←list"))
	return strings.Join(lines, "\n")
}

func (v TasksView) renderTaskFormInline() []string {
	fl := func(idx int, label string) string {
		if v.tForm.focusIdx == idx {
			return sTitle.Render("> " + label)
		}
		return sMuted.Render("  " + label)
	}
	return []string{
		sTitle.Render("── New task ──"),
		fl(0, "Title:    ") + " " + v.tForm.title.View(),
		fl(1, "Context:  ") + " " + v.tForm.contextDisplay(v.tForm.focusIdx == 1),
		fl(2, "Labels:   ") + " " + v.tForm.labels.View(),
		fl(3, "Priority: ") + " " + v.tForm.priority.View(),
		sMuted.Render("  tab next  ←/→ context  ctrl+s save  esc cancel"),
		"",
	}
}

func (v TasksView) renderTaskFormDetail(t model.Task, width int) string {
	return renderTaskEditFormShared(v.tForm, t.ID, width)
}

func (v TasksView) renderCarryPrompt(t model.Task, width int) string {
	raw := strings.TrimSpace(v.carryInput.Value())
	hint := ""
	if _, err := time.Parse(model.DateFormat, raw); err != nil && raw != "" {
		hint = sCarried.Render("  invalid date")
	}

	return strings.Join([]string{
		sTitle.Render("Carry task to…"),
		sMuted.Render("→ " + t.Title),
		strings.Repeat("─", min(width, 36)),
		"",
		"  Date: " + v.carryInput.View() + hint,
		"",
		sMuted.Render("  enter to confirm  esc to cancel"),
	}, "\n")
}

func (v TasksView) renderNotesEditor(t model.Task, width int) string {
	return renderNotesEditorShared(v.notesArea, t.Title, width)
}

func (v TasksView) renderLogForm(t model.Task, width int) string {
	return renderLogFormShared(v.logForm, t.Title, width)
}

func (v TasksView) renderActionsEditor(t model.Task, width int) string {
	return renderActionsShared(t.Actions, v.actionCursor, v.actionAdding, v.actionInput, width)
}

func (v TasksView) Date() time.Time    { return v.date }
func (v TasksView) Contexts() []string { return v.contexts }

func (v TasksView) showingDone() bool {
	for _, s := range v.filter.Statuses {
		if s == model.StatusDone {
			return true
		}
	}
	return false
}

func (v TasksView) toggleShowDone() (TasksView, tea.Cmd) {
	if v.showingDone() {
		filtered := v.filter.Statuses[:0:0]
		for _, s := range v.filter.Statuses {
			if s != model.StatusDone {
				filtered = append(filtered, s)
			}
		}
		v.filter.Statuses = filtered
	} else {
		v.filter.Statuses = append(v.filter.Statuses, model.StatusDone)
	}
	v.cursor = 0
	// Re-run global query if in global mode so statuses are reflected
	if v.globalResults != nil {
		return v, func() tea.Msg {
			sf := storage.TaskFilter{
				Labels:   v.filter.Labels,
				Search:   v.filter.Search,
				Statuses: v.filter.Statuses,
			}
			tasks, err := v.store.ListTasks(sf)
			if err != nil {
				return errMsg{err}
			}
			return GlobalTasksMsg{Tasks: tasks}
		}
	}
	return v, nil
}

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
