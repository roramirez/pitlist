package model

import "time"

type TaskStatus string
type Priority string

const (
	DateFormat     = "2006-01-02"
	DateTimeFormat = "2006-01-02T15:04"
)

const (
	StatusTodo       TaskStatus = "todo"
	StatusInProgress TaskStatus = "in_progress"
	StatusDone       TaskStatus = "done"
	StatusCancelled  TaskStatus = "cancelled"
)

const (
	PriorityLow    Priority = "low"
	PriorityMedium Priority = "medium"
	PriorityHigh   Priority = "high"
)

type ActivityRef struct {
	ID   string `yaml:"id"`
	Date string `yaml:"date"` // YYYY-MM-DD
}

// Action is a single checklist step inside a Task.
type Action struct {
	ID    string `yaml:"id"`
	Title string `yaml:"title"`
	Done  bool   `yaml:"done"`
}

type Task struct {
	ID           string        `yaml:"id"`
	Title        string        `yaml:"title"`
	Context      string        `yaml:"context,omitempty"` // e.g. work, personal, other
	Notes        string        `yaml:"notes,omitempty"`
	Labels       []string      `yaml:"labels,omitempty"`
	Status       TaskStatus    `yaml:"status"`
	Priority     Priority      `yaml:"priority"`
	CreatedAt    time.Time     `yaml:"created_at"`
	UpdatedAt    time.Time     `yaml:"updated_at"`
	DoneAt       *time.Time    `yaml:"done_at,omitempty"`
	DueDate      string        `yaml:"due_date,omitempty"`
	CarryFrom    string        `yaml:"carry_from,omitempty"`
	CarryTo      string        `yaml:"carry_to,omitempty"`
	ActivityRefs []ActivityRef `yaml:"activity_refs,omitempty"`
	Actions      []Action      `yaml:"actions,omitempty"`
}

// CloneSkeleton returns a fresh, undone copy of the task: status reset to todo,
// done/carry/activity metadata cleared, and every action copied with Done=false.
// ID, CreatedAt and UpdatedAt are left zero for the caller to assign.
func (t Task) CloneSkeleton() Task {
	clone := t
	clone.ID = ""
	clone.Status = StatusTodo
	clone.DoneAt = nil
	clone.CarryFrom = ""
	clone.CarryTo = ""
	clone.ActivityRefs = nil
	clone.CreatedAt = time.Time{}
	clone.UpdatedAt = time.Time{}
	clone.Labels = append([]string(nil), t.Labels...)
	if len(t.Actions) == 0 {
		clone.Actions = nil
		return clone
	}
	clone.Actions = make([]Action, len(t.Actions))
	for i, a := range t.Actions {
		a.Done = false
		clone.Actions[i] = a
	}
	return clone
}

type DayPlan struct {
	Date  time.Time `yaml:"date"`
	Tasks []Task    `yaml:"tasks"`
}

type FutureList struct {
	Tasks []Task `yaml:"tasks"`
}
