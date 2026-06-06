package views

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/roramirez/pitlist/internal/model"
)

const actionIDFmt = "ac-%03d"

// nextActionID returns the next action ID for a task's action slice.
func nextActionID(actions []model.Action) string {
	return fmt.Sprintf(actionIDFmt, len(actions)+1)
}

// doneCount returns the number of completed actions.
func doneCount(actions []model.Action) int {
	n := 0
	for _, a := range actions {
		if a.Done {
			n++
		}
	}
	return n
}

// actionBadge returns a "[done/total]" string, or "" when actions is empty.
func actionBadge(actions []model.Action) string {
	if len(actions) == 0 {
		return ""
	}
	return fmt.Sprintf(" [%d/%d]", doneCount(actions), len(actions))
}

// actionEditorResult holds the outcome of a single key press in actions mode.
type actionEditorResult struct {
	cursor   int
	adding   bool
	input    textinput.Model
	exitMode bool
	toggleID string
	deleteID string
	newTitle string
	blink    bool
}

// handleActionEditorKey processes a key event in actions-editor mode and returns
// what the caller should do next. It is pure (no side effects, no store access).
func handleActionEditorKey(cursor int, adding bool, input textinput.Model, actions []model.Action, msg tea.KeyMsg) actionEditorResult {
	res := actionEditorResult{cursor: cursor, adding: adding, input: input}
	if adding {
		return handleActionAddKey(res, msg)
	}
	return handleActionNavKey(res, actions, msg)
}

// handleActionAddKey handles keys when the inline add-input is active.
func handleActionAddKey(res actionEditorResult, msg tea.KeyMsg) actionEditorResult {
	switch msg.String() {
	case "esc":
		res.adding = false
		res.input.Blur()
	case "enter", "ctrl+s":
		res.newTitle = strings.TrimSpace(res.input.Value())
		res.adding = false
		res.input.Blur()
	default:
		res.input, _ = res.input.Update(msg)
	}
	return res
}

// handleActionNavKey handles navigation and action-management keys.
func handleActionNavKey(res actionEditorResult, actions []model.Action, msg tea.KeyMsg) actionEditorResult {
	switch msg.String() {
	case "esc":
		res.exitMode = true
	case "a":
		res.adding = true
		res.input.Reset()
		res.input.Focus()
		res.blink = true
	case "j", "down":
		res.cursor = min(res.cursor+1, max(len(actions)-1, 0))
	case "k", "up":
		res.cursor = max(res.cursor-1, 0)
	case " ":
		res.toggleID = actionIDAt(actions, res.cursor)
	case "D":
		res.deleteID = actionIDAt(actions, res.cursor)
	}
	return res
}

// actionIDAt returns the ID of the action at cursor, or "" if out of range.
func actionIDAt(actions []model.Action, cursor int) string {
	if cursor < len(actions) {
		return actions[cursor].ID
	}
	return ""
}

// applyActionAdd appends a new action to the matching task in the slice.
func applyActionAdd(tasks []model.Task, taskID, title string) []model.Task {
	for i := range tasks {
		if tasks[i].ID == taskID {
			tasks[i].Actions = append(tasks[i].Actions, model.Action{
				ID:    nextActionID(tasks[i].Actions),
				Title: title,
			})
			tasks[i].UpdatedAt = time.Now().UTC()
			break
		}
	}
	return tasks
}

// applyActionToggle flips Done on the matching action inside the matching task.
func applyActionToggle(tasks []model.Task, taskID, actionID string) []model.Task {
	for i := range tasks {
		if tasks[i].ID == taskID {
			for j := range tasks[i].Actions {
				if tasks[i].Actions[j].ID == actionID {
					tasks[i].Actions[j].Done = !tasks[i].Actions[j].Done
					tasks[i].UpdatedAt = time.Now().UTC()
					break
				}
			}
			break
		}
	}
	return tasks
}

// applyActionDelete removes the matching action from the matching task.
func applyActionDelete(tasks []model.Task, taskID, actionID string) []model.Task {
	for i := range tasks {
		if tasks[i].ID == taskID {
			remaining := make([]model.Action, 0, len(tasks[i].Actions))
			for _, a := range tasks[i].Actions {
				if a.ID != actionID {
					remaining = append(remaining, a)
				}
			}
			tasks[i].Actions = remaining
			tasks[i].UpdatedAt = time.Now().UTC()
			break
		}
	}
	return tasks
}

// renderActionsDetailSection renders the summary checklist for the normal detail pane.
// Returns nil when the task has no actions.
func renderActionsDetailSection(actions []model.Action) []string {
	if len(actions) == 0 {
		return nil
	}
	lines := []string{"", sTitle.Render("Actions:" + actionBadge(actions))}
	for _, a := range actions {
		check := "[ ]"
		if a.Done {
			check = "[x]"
		}
		lines = append(lines, fmt.Sprintf("  %s %s", check, a.Title))
	}
	return lines
}

// renderActionsShared renders the actions editor/viewer used by TasksView and FutureView.
func renderActionsShared(actions []model.Action, cursor int, adding bool, input textinput.Model, width int) string {
	header := sTitle.Render("Actions" + actionBadge(actions))
	lines := []string{header, strings.Repeat("─", min(width, 36)), ""}
	lines = append(lines, renderActionRows(actions, cursor)...)
	if adding {
		lines = append(lines, "  + "+input.View())
	}
	lines = append(lines, "", sMuted.Render("j/k navigate  space toggle  a add  D delete  esc done"))
	return strings.Join(lines, "\n")
}

// renderActionRows renders each action as a check-box line.
func renderActionRows(actions []model.Action, cursor int) []string {
	if len(actions) == 0 {
		return []string{sMuted.Render("  No actions yet. Press 'a' to add one.")}
	}
	rows := make([]string, len(actions))
	for i, a := range actions {
		check := "[ ]"
		if a.Done {
			check = "[x]"
		}
		line := fmt.Sprintf("  %s %s", check, a.Title)
		if i == cursor {
			line = sSelected.Render(line)
		}
		rows[i] = line
	}
	return rows
}

// applyTaskFormFocus blurs non-active task form inputs and focuses the active one.
// Returns the updated form and whether a blink cmd is needed.
func applyTaskFormFocus(f taskForm) (taskForm, bool) {
	f.title.Blur()
	f.labels.Blur()
	f.priority.Blur()
	switch f.focusIdx {
	case 0:
		f.title.Focus()
		return f, true
	case 1:
		return f, false
	case 2:
		f.labels.Focus()
		return f, true
	case 3:
		f.priority.Focus()
		return f, true
	}
	return f, false
}

// blurAllLogFields blurs every input in a quickLogForm.
func blurAllLogFields(f quickLogForm) quickLogForm {
	f.desc.Blur()
	f.tags.Blur()
	f.duration.Blur()
	f.dateInput.Blur()
	return f
}

// focusActiveLogField focuses whichever field matches f.focusIdx.
func focusActiveLogField(f quickLogForm) quickLogForm {
	switch f.focusIdx {
	case 0:
		f.desc.Focus()
	case 1:
		f.tags.Focus()
	case 2:
		f.duration.Focus()
	case 3:
		f.dateInput.Focus()
	}
	return f
}

// updateLogFormField routes a key event to the focused field of a quickLogForm.
func updateLogFormField(f quickLogForm, msg tea.KeyMsg) (quickLogForm, tea.Cmd) {
	var cmd tea.Cmd
	switch f.focusIdx {
	case 0:
		f.desc, cmd = f.desc.Update(msg)
	case 1:
		f.tags, cmd = f.tags.Update(msg)
	case 2:
		f.duration, cmd = f.duration.Update(msg)
	case 3:
		f.dateInput, cmd = f.dateInput.Update(msg)
	}
	return f, cmd
}

const (
	sentinelCtx = "\x00" // forces first context header on init
	carriedCtx  = "carried"
)

// handleTaskFormKey routes a key event to the correct taskForm field.
func handleTaskFormKey(f taskForm, msg tea.KeyMsg) taskForm {
	n := len(f.contexts)
	switch f.focusIdx {
	case 0:
		f.title, _ = f.title.Update(msg)
	case 1:
		switch msg.String() {
		case "left", "h":
			if n > 0 {
				f.contextIdx = (f.contextIdx-1+n+1)%(n+1) - 1
			}
		case "right", "l":
			if n > 0 {
				f.contextIdx++
				if f.contextIdx >= n {
					f.contextIdx = -1
				}
			}
		}
	case 2:
		f.labels, _ = f.labels.Update(msg)
	case 3:
		f.priority, _ = f.priority.Update(msg)
	}
	return f
}

// contextSectionHeader renders a separator line for a context group.
func contextSectionHeader(ctx string, width int) string {
	label := ctx
	if label == "" {
		label = "—"
	}
	sep := strings.Repeat("─", max(0, width-len(label)-4))
	return sTitle.Render("  "+label) + "  " + sMuted.Render(sep)
}

// renderLogFormShared renders the quick-log activity form used by TasksView and FutureView.
func renderLogFormShared(f quickLogForm, taskTitle string, width int) string {
	fl := func(idx int, label string) string {
		if f.focusIdx == idx {
			return sTitle.Render("> " + label)
		}
		return sMuted.Render("  " + label)
	}
	return strings.Join([]string{
		sTitle.Render("Log activity"),
		sMuted.Render("→ " + taskTitle),
		strings.Repeat("─", min(width, 36)),
		"",
		fl(0, "Description: ") + " " + f.desc.View(),
		fl(1, "Tags:        ") + " " + f.tags.View(),
		fl(2, "Minutes:     ") + " " + f.duration.View(),
		fl(3, "Date:        ") + " " + f.dateInput.View(),
		sMuted.Render("  Task ref:   " + f.taskID),
		"",
		sMuted.Render("tab next  ctrl+s save  esc cancel"),
	}, "\n")
}

// renderTaskEditFormShared renders the edit-task form used by TasksView and FutureView.
func renderTaskEditFormShared(f taskForm, taskID string, width int) string {
	fl := func(idx int, label string) string {
		if f.focusIdx == idx {
			return sTitle.Render("> " + label)
		}
		return sMuted.Render("  " + label)
	}
	return strings.Join([]string{
		sTitle.Render("Edit task"),
		sMuted.Render("ID: " + taskID),
		strings.Repeat("─", min(width, 36)),
		"",
		fl(0, "Title:    ") + " " + f.title.View(),
		fl(1, "Context:  ") + " " + f.contextDisplay(f.focusIdx == 1),
		fl(2, "Labels:   ") + " " + f.labels.View(),
		fl(3, "Priority: ") + " " + f.priority.View(),
		"",
		sMuted.Render("  tab next  ←/→ context  ctrl+s save  esc cancel"),
	}, "\n")
}

// renderNotesEditorShared renders the notes textarea used by TasksView and FutureView.
func renderNotesEditorShared(area textarea.Model, taskTitle string, width int) string {
	area.SetWidth(width - 2)
	area.SetHeight(10)
	return strings.Join([]string{
		sTitle.Render("Notes: " + taskTitle),
		strings.Repeat("─", min(len(taskTitle)+8, width)),
		"",
		area.View(),
		"",
		sMuted.Render("ctrl+s save  esc cancel"),
	}, "\n")
}

// renderTaskHeader renders the title, separator, and metadata lines common to all detail panes.
func renderTaskHeader(t model.Task, width int) []string {
	lines := []string{
		sTitle.Render(t.Title),
		strings.Repeat("─", min(len(t.Title)+2, width)),
		"",
	}
	if t.Context != "" {
		lines = append(lines, fmt.Sprintf("Context:  %s", t.Context))
	}
	lines = append(lines, fmt.Sprintf("Status:   %s", t.Status))
	lines = append(lines, fmt.Sprintf("Priority: %s", t.Priority))
	if len(t.Labels) > 0 {
		lines = append(lines, fmt.Sprintf("Labels:   %s", strings.Join(t.Labels, "  ")))
	}
	return lines
}

// renderLinkedActivities renders the activity summary section for a detail pane.
func renderLinkedActivities(linked []model.ActivityEntry) []string {
	if len(linked) == 0 {
		return nil
	}
	totalMin := 0
	for _, e := range linked {
		totalMin += e.DurationMin
	}
	header := sTitle.Render("Activity:")
	if totalMin > 0 {
		total := fmt.Sprintf("%dh %02dm", totalMin/60, totalMin%60)
		header += "  " + sCarried.Render("∑ "+total)
	}
	lines := []string{"", header}
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
	return lines
}
