package model

import "time"

type ActivityEntry struct {
	ID          string    `yaml:"id"`
	Timestamp   time.Time `yaml:"timestamp"`
	Description string    `yaml:"description"`
	Tags        []string  `yaml:"tags,omitempty"`
	TaskRef     string    `yaml:"task_ref,omitempty"`
	DurationMin int       `yaml:"duration_min,omitempty"`
}

type ActivityLog struct {
	Date    time.Time       `yaml:"date"`
	Entries []ActivityEntry `yaml:"entries"`
}
