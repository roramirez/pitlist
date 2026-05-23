package demo

import (
	"os"
	"time"

	"github.com/roramirez/pitlist/internal/model"
	"github.com/roramirez/pitlist/internal/storage"
)

// Seed creates a temporary data directory pre-populated with demo tasks and
// activity entries. The caller is responsible for removing the directory.
func Seed() (dataDir string, cleanup func(), err error) {
	dir, err := os.MkdirTemp("", "pitlist-demo-*")
	if err != nil {
		return "", nil, err
	}
	cleanup = func() { os.RemoveAll(dir) }

	store, err := storage.NewYAMLStore(dir)
	if err != nil {
		cleanup()
		return "", nil, err
	}

	if err := SeedInto(store); err != nil {
		cleanup()
		return "", nil, err
	}
	return dir, cleanup, nil
}

// SeedInto populates an existing store with demo data.
func SeedInto(store *storage.YAMLStore) error {
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
	yesterday := today.AddDate(0, 0, -1)
	tomorrow := today.AddDate(0, 0, 1)

	if err := seedTasks(store, today, yesterday, tomorrow); err != nil {
		return err
	}
	return seedActivity(store, today, yesterday)
}

func seedTasks(store *storage.YAMLStore, today, yesterday, tomorrow time.Time) error {
	todayPlan := &model.DayPlan{
		Date: today,
		Tasks: []model.Task{
			{
				ID:        "t-demo-001",
				Title:     "Review authentication middleware refactor",
				Context:   "work",
				Labels:    []string{"backend", "security"},
				Status:    model.StatusInProgress,
				Priority:  model.PriorityHigh,
				Notes:     "Check that token expiry edge cases are handled correctly.",
				CreatedAt: today.Add(8 * time.Hour),
				UpdatedAt: today.Add(9 * time.Hour),
			},
			{
				ID:        "t-demo-002",
				Title:     "Write unit tests for user service",
				Context:   "work",
				Labels:    []string{"testing", "backend"},
				Status:    model.StatusTodo,
				Priority:  model.PriorityMedium,
				DueDate:   tomorrow.Format(model.DateFormat),
				CreatedAt: today.Add(8*time.Hour + 30*time.Minute),
				UpdatedAt: today.Add(8*time.Hour + 30*time.Minute),
			},
			{
				ID:        "t-demo-003",
				Title:     "Update API documentation",
				Context:   "work",
				Labels:    []string{"docs"},
				Status:    model.StatusTodo,
				Priority:  model.PriorityLow,
				CreatedAt: today.Add(9 * time.Hour),
				UpdatedAt: today.Add(9 * time.Hour),
			},
			{
				ID:        "t-demo-004",
				Title:     "Fix flaky integration test in CI",
				Context:   "work",
				Labels:    []string{"testing", "ci"},
				Status:    model.StatusDone,
				Priority:  model.PriorityHigh,
				CreatedAt: today.Add(7 * time.Hour),
				UpdatedAt: today.Add(10 * time.Hour),
				ActivityRefs: []model.ActivityRef{
					{ID: "a-demo-001", Date: today.Format(model.DateFormat)},
				},
			},
			{
				ID:        "t-demo-005",
				Title:     "Buy groceries",
				Context:   "personal",
				Labels:    []string{"errands"},
				Status:    model.StatusTodo,
				Priority:  model.PriorityLow,
				CreatedAt: today.Add(8 * time.Hour),
				UpdatedAt: today.Add(8 * time.Hour),
			},
			{
				ID:        "t-demo-008",
				Title:     "Call dentist to schedule appointment",
				Context:   "personal",
				Labels:    []string{"health"},
				Status:    model.StatusTodo,
				Priority:  model.PriorityMedium,
				DueDate:   tomorrow.Format(model.DateFormat),
				CreatedAt: today.Add(8*time.Hour + 15*time.Minute),
				UpdatedAt: today.Add(8*time.Hour + 15*time.Minute),
			},
			{
				ID:        "t-demo-009",
				Title:     "Read chapter 4 of Designing Data-Intensive Applications",
				Context:   "personal",
				Labels:    []string{"learning"},
				Status:    model.StatusInProgress,
				Priority:  model.PriorityLow,
				Notes:     "Stopped at section on LSM trees vs B-trees.",
				CreatedAt: today.Add(7*time.Hour + 30*time.Minute),
				UpdatedAt: today.Add(7*time.Hour + 30*time.Minute),
			},
			{
				ID:        "t-demo-010",
				Title:     "30 min run",
				Context:   "personal",
				Labels:    []string{"fitness"},
				Status:    model.StatusDone,
				Priority:  model.PriorityMedium,
				CreatedAt: today.Add(6 * time.Hour),
				UpdatedAt: today.Add(6*time.Hour + 35*time.Minute),
				ActivityRefs: []model.ActivityRef{
					{ID: "a-demo-005", Date: today.Format(model.DateFormat)},
				},
			},
		},
	}

	yesterdayPlan := &model.DayPlan{
		Date: yesterday,
		Tasks: []model.Task{
			{
				ID:       "t-demo-006",
				Title:    "Deploy staging environment",
				Context:  "work",
				Labels:   []string{"devops"},
				Status:   model.StatusDone,
				Priority: model.PriorityHigh,
				ActivityRefs: []model.ActivityRef{
					{ID: "a-demo-003", Date: yesterday.Format(model.DateFormat)},
				},
				CreatedAt: yesterday.Add(9 * time.Hour),
				UpdatedAt: yesterday.Add(14 * time.Hour),
			},
			{
				ID:        "t-demo-007",
				Title:     "Research new caching strategy",
				Context:   "work",
				Labels:    []string{"backend", "performance"},
				Status:    model.StatusDone,
				Priority:  model.PriorityMedium,
				CreatedAt: yesterday.Add(10 * time.Hour),
				UpdatedAt: yesterday.Add(16 * time.Hour),
			},
			{
				ID:        "t-demo-011",
				Title:     "Meal prep for the week",
				Context:   "personal",
				Labels:    []string{"health", "errands"},
				Status:    model.StatusDone,
				Priority:  model.PriorityMedium,
				CreatedAt: yesterday.Add(8 * time.Hour),
				UpdatedAt: yesterday.Add(19 * time.Hour),
				ActivityRefs: []model.ActivityRef{
					{ID: "a-demo-005", Date: yesterday.Format(model.DateFormat)},
				},
			},
			{
				ID:        "t-demo-012",
				Title:     "Catch up with Ana",
				Context:   "personal",
				Labels:    []string{"social"},
				Status:    model.StatusDone,
				Priority:  model.PriorityLow,
				CreatedAt: yesterday.Add(9 * time.Hour),
				UpdatedAt: yesterday.Add(21 * time.Hour),
			},
		},
	}

	if err := store.SaveDayPlan(todayPlan); err != nil {
		return err
	}
	return store.SaveDayPlan(yesterdayPlan)
}

func seedActivity(store *storage.YAMLStore, today, yesterday time.Time) error {
	todayLog := &model.ActivityLog{
		Date: today,
		Entries: []model.ActivityEntry{
			{
				ID:          "a-demo-001",
				Timestamp:   today.Add(10 * time.Hour),
				Description: "Investigated flaky test — race condition in DB teardown",
				Tags:        []string{"debugging", "ci"},
				TaskRef:     "t-demo-004",
				DurationMin: 45,
			},
			{
				ID:          "a-demo-002",
				Timestamp:   today.Add(11 * time.Hour),
				Description: "Code review for PR #87 — payment service refactor",
				Tags:        []string{"review"},
				DurationMin: 30,
			},
			{
				ID:          "a-demo-003",
				Timestamp:   today.Add(14 * time.Hour),
				Description: "Team sync — discussed Q3 roadmap priorities",
				Tags:        []string{"meeting"},
				DurationMin: 60,
			},
			{
				ID:          "a-demo-005",
				Timestamp:   today.Add(6 * time.Hour),
				Description: "Morning run — 5.2 km, felt good",
				Tags:        []string{"fitness"},
				TaskRef:     "t-demo-010",
				DurationMin: 35,
			},
			{
				ID:          "a-demo-006",
				Timestamp:   today.Add(22 * time.Hour),
				Description: "Read DDIA ch4 — LSM trees vs B-trees, good notes",
				Tags:        []string{"learning"},
				TaskRef:     "t-demo-009",
				DurationMin: 50,
			},
		},
	}

	yesterdayLog := &model.ActivityLog{
		Date: yesterday,
		Entries: []model.ActivityEntry{
			{
				ID:          "a-demo-003",
				Timestamp:   yesterday.Add(10 * time.Hour),
				Description: "Deployed new staging infra — updated Terraform modules",
				Tags:        []string{"devops", "infra"},
				TaskRef:     "t-demo-006",
				DurationMin: 90,
			},
			{
				ID:          "a-demo-004",
				Timestamp:   yesterday.Add(15 * time.Hour),
				Description: "Read Redis caching patterns article, notes in Notion",
				Tags:        []string{"learning"},
				DurationMin: 40,
			},
			{
				ID:          "a-demo-005",
				Timestamp:   yesterday.Add(18*time.Hour + 30*time.Minute),
				Description: "Meal prep — rice, roasted veggies, chicken for 4 days",
				Tags:        []string{"health"},
				TaskRef:     "t-demo-011",
				DurationMin: 75,
			},
			{
				ID:          "a-demo-006",
				Timestamp:   yesterday.Add(21 * time.Hour),
				Description: "Called Ana, caught up for ~1h — she's moving to Berlin",
				Tags:        []string{"social"},
				DurationMin: 60,
			},
		},
	}

	if err := store.SaveActivityLog(todayLog); err != nil {
		return err
	}
	return store.SaveActivityLog(yesterdayLog)
}
