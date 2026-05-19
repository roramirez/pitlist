package storage

import (
	"time"

	"github.com/roramirez/pitlist/internal/model"
)

type TaskFilter struct {
	Labels   []string
	Statuses []model.TaskStatus
	From     *time.Time
	To       *time.Time
	Search   string
}

type ActivityFilter struct {
	Tags    []string
	From    *time.Time
	To      *time.Time
	Search  string
	TaskRef string
}

type Store interface {
	GetDayPlan(date time.Time) (*model.DayPlan, error)
	SaveDayPlan(plan *model.DayPlan) error
	GetTaskByID(id string) (*model.Task, time.Time, error)
	ListTasks(filter TaskFilter) ([]*model.Task, error)

	GetActivityLog(date time.Time) (*model.ActivityLog, error)
	SaveActivityLog(log *model.ActivityLog) error
	ListActivity(filter ActivityFilter) ([]*model.ActivityEntry, error)
	GetActivitiesByRefs(refs []model.ActivityRef, fallbackDate time.Time) ([]*model.ActivityEntry, error)
	AddActivityRefToTask(taskID string, ref model.ActivityRef) error
}
