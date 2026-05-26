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
}

type DayPlan struct {
	Date  time.Time `yaml:"date"`
	Tasks []Task    `yaml:"tasks"`
}

type FutureList struct {
	Tasks []Task `yaml:"tasks"`
}
