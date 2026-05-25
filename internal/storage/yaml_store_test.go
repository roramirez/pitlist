package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/roramirez/pitlist/internal/model"
)

func newTestStore(t *testing.T) *YAMLStore {
	t.Helper()
	s, err := NewYAMLStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewYAMLStore: %v", err)
	}
	return s
}

func TestDayPlanRoundTrip(t *testing.T) {
	s := newTestStore(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)

	plan := &model.DayPlan{
		Date: date,
		Tasks: []model.Task{
			{
				ID:        "t-20260518-001",
				Title:     "Test task",
				Labels:    []string{"work"},
				Status:    model.StatusTodo,
				Priority:  model.PriorityHigh,
				CreatedAt: date,
				UpdatedAt: date,
			},
		},
	}

	if err := s.SaveDayPlan(plan); err != nil {
		t.Fatalf("SaveDayPlan: %v", err)
	}

	got, err := s.GetDayPlan(date)
	if err != nil {
		t.Fatalf("GetDayPlan: %v", err)
	}

	if len(got.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(got.Tasks))
	}
	if got.Tasks[0].Title != "Test task" {
		t.Errorf("title mismatch: got %q", got.Tasks[0].Title)
	}
	if got.Tasks[0].Status != model.StatusTodo {
		t.Errorf("status mismatch: got %q", got.Tasks[0].Status)
	}
}

func TestGetDayPlanMissing(t *testing.T) {
	s := newTestStore(t)
	date := time.Date(2026, 1, 1, 0, 0, 0, 0, time.UTC)
	plan, err := s.GetDayPlan(date)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(plan.Tasks) != 0 {
		t.Errorf("expected empty plan, got %d tasks", len(plan.Tasks))
	}
}

func TestGetTaskByID(t *testing.T) {
	s := newTestStore(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	plan := &model.DayPlan{
		Date: date,
		Tasks: []model.Task{
			{ID: "t-20260518-001", Title: "Find me", Status: model.StatusTodo, CreatedAt: date, UpdatedAt: date},
		},
	}
	if err := s.SaveDayPlan(plan); err != nil {
		t.Fatal(err)
	}

	task, _, err := s.GetTaskByID("t-20260518-001")
	if err != nil {
		t.Fatalf("GetTaskByID: %v", err)
	}
	if task.Title != "Find me" {
		t.Errorf("unexpected title: %q", task.Title)
	}
}

func TestListTasksFilterByLabel(t *testing.T) {
	s := newTestStore(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	plan := &model.DayPlan{
		Date: date,
		Tasks: []model.Task{
			{ID: "t-20260518-001", Title: "Work task", Labels: []string{"work"}, Status: model.StatusTodo, CreatedAt: date, UpdatedAt: date},
			{ID: "t-20260518-002", Title: "Personal task", Labels: []string{"personal"}, Status: model.StatusTodo, CreatedAt: date, UpdatedAt: date},
		},
	}
	if err := s.SaveDayPlan(plan); err != nil {
		t.Fatal(err)
	}

	results, err := s.ListTasks(TaskFilter{Labels: []string{"work"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 {
		t.Fatalf("expected 1, got %d", len(results))
	}
	if results[0].Title != "Work task" {
		t.Errorf("unexpected task: %q", results[0].Title)
	}
}

func TestListTasksFilterByStatus(t *testing.T) {
	s := newTestStore(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	plan := &model.DayPlan{
		Date: date,
		Tasks: []model.Task{
			{ID: "t-20260518-001", Title: "Todo", Status: model.StatusTodo, CreatedAt: date, UpdatedAt: date},
			{ID: "t-20260518-002", Title: "Done", Status: model.StatusDone, CreatedAt: date, UpdatedAt: date},
		},
	}
	if err := s.SaveDayPlan(plan); err != nil {
		t.Fatal(err)
	}

	results, err := s.ListTasks(TaskFilter{Statuses: []model.TaskStatus{model.StatusDone}})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Title != "Done" {
		t.Errorf("unexpected results: %v", results)
	}
}

func TestActivityLogRoundTrip(t *testing.T) {
	s := newTestStore(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	ts := time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC)

	log := &model.ActivityLog{
		Date: date,
		Entries: []model.ActivityEntry{
			{
				ID:          "a-20260518-001",
				Timestamp:   ts,
				Description: "Did something useful",
				Tags:        []string{"work"},
				DurationMin: 30,
			},
		},
	}

	if err := s.SaveActivityLog(log); err != nil {
		t.Fatal(err)
	}

	got, err := s.GetActivityLog(date)
	if err != nil {
		t.Fatal(err)
	}
	if len(got.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(got.Entries))
	}
	if got.Entries[0].Description != "Did something useful" {
		t.Errorf("description mismatch: %q", got.Entries[0].Description)
	}
}

func TestListActivityFilterByTag(t *testing.T) {
	s := newTestStore(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	ts := time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC)

	log := &model.ActivityLog{
		Date: date,
		Entries: []model.ActivityEntry{
			{ID: "a-20260518-001", Timestamp: ts, Description: "Debugging", Tags: []string{"debugging"}},
			{ID: "a-20260518-002", Timestamp: ts, Description: "Meeting", Tags: []string{"meetings"}},
		},
	}
	if err := s.SaveActivityLog(log); err != nil {
		t.Fatal(err)
	}

	results, err := s.ListActivity(ActivityFilter{Tags: []string{"debugging"}})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Description != "Debugging" {
		t.Errorf("unexpected results: %v", results)
	}
}

func TestNextIDs(t *testing.T) {
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	plan := &model.DayPlan{Date: date, Tasks: []model.Task{}}
	id := NextTaskID(plan)
	if id != "t-20260518-001" {
		t.Errorf("unexpected task ID: %q", id)
	}

	log := &model.ActivityLog{Date: date, Entries: []model.ActivityEntry{}}
	aid := NextActivityID(log)
	if aid != "a-20260518-001" {
		t.Errorf("unexpected activity ID: %q", aid)
	}
}

func TestNextIDsIncrement(t *testing.T) {
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	plan := &model.DayPlan{
		Date: date,
		Tasks: []model.Task{
			{ID: "t-20260518-001"},
			{ID: "t-20260518-002"},
		},
	}
	id := NextTaskID(plan)
	if id != "t-20260518-003" {
		t.Errorf("unexpected task ID: %q", id)
	}
}

func TestListTasksDateRange(t *testing.T) {
	s := newTestStore(t)
	d1 := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC)
	d3 := time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC)

	for i, d := range []time.Time{d1, d2, d3} {
		plan := &model.DayPlan{
			Date: d,
			Tasks: []model.Task{
				{
					ID:     fmt.Sprintf("t-%s-001", d.Format("20060102")),
					Title:  fmt.Sprintf("Task %d", i+1),
					Status: model.StatusTodo, CreatedAt: d, UpdatedAt: d,
				},
			},
		}
		if err := s.SaveDayPlan(plan); err != nil {
			t.Fatal(err)
		}
	}

	from := d2
	results, err := s.ListTasks(TaskFilter{From: &from})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2, got %d", len(results))
	}

	to := d2
	results, err = s.ListTasks(TaskFilter{To: &to})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2, got %d", len(results))
	}

	results, err = s.ListTasks(TaskFilter{From: &d2, To: &d2})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Title != "Task 2" {
		t.Fatalf("expected Task 2, got %v", results)
	}
}

func TestListTasksSearchFilter(t *testing.T) {
	s := newTestStore(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	plan := &model.DayPlan{
		Date: date,
		Tasks: []model.Task{
			{ID: "t-20260518-001", Title: "Refactor auth module", Status: model.StatusTodo, CreatedAt: date, UpdatedAt: date},
			{ID: "t-20260518-002", Title: "Write docs", Status: model.StatusTodo, CreatedAt: date, UpdatedAt: date},
		},
	}
	if err := s.SaveDayPlan(plan); err != nil {
		t.Fatal(err)
	}

	results, err := s.ListTasks(TaskFilter{Search: "auth"})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Title != "Refactor auth module" {
		t.Fatalf("unexpected results: %v", results)
	}
}

func TestListActivityDateRange(t *testing.T) {
	s := newTestStore(t)
	d1 := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC)
	d3 := time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC)

	for i, d := range []time.Time{d1, d2, d3} {
		al := &model.ActivityLog{
			Date: d,
			Entries: []model.ActivityEntry{
				{
					ID:          fmt.Sprintf("a-%s-001", d.Format("20060102")),
					Timestamp:   d,
					Description: fmt.Sprintf("Activity %d", i+1),
				},
			},
		}
		if err := s.SaveActivityLog(al); err != nil {
			t.Fatal(err)
		}
	}

	from := d2
	results, err := s.ListActivity(ActivityFilter{From: &from})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2, got %d", len(results))
	}

	to := d2
	results, err = s.ListActivity(ActivityFilter{To: &to})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2, got %d", len(results))
	}
}

func TestListActivitySearchFilter(t *testing.T) {
	s := newTestStore(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	al := &model.ActivityLog{
		Date: date,
		Entries: []model.ActivityEntry{
			{ID: "a-20260518-001", Timestamp: date, Description: "Debugged the login flow"},
			{ID: "a-20260518-002", Timestamp: date, Description: "Team standup"},
		},
	}
	if err := s.SaveActivityLog(al); err != nil {
		t.Fatal(err)
	}

	results, err := s.ListActivity(ActivityFilter{Search: "login"})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Description != "Debugged the login flow" {
		t.Fatalf("unexpected results: %v", results)
	}
}

func TestListActivityTaskRefFilter(t *testing.T) {
	s := newTestStore(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	al := &model.ActivityLog{
		Date: date,
		Entries: []model.ActivityEntry{
			{ID: "a-20260518-001", Timestamp: date, Description: "Work on task", TaskRef: "t-20260518-001"},
			{ID: "a-20260518-002", Timestamp: date, Description: "Unrelated work"},
		},
	}
	if err := s.SaveActivityLog(al); err != nil {
		t.Fatal(err)
	}

	results, err := s.ListActivity(ActivityFilter{TaskRef: "t-20260518-001"})
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].TaskRef != "t-20260518-001" {
		t.Fatalf("unexpected results: %v", results)
	}
}

func TestGetActivitiesByRefs(t *testing.T) {
	s := newTestStore(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	ts := time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC)

	al := &model.ActivityLog{
		Date: date,
		Entries: []model.ActivityEntry{
			{ID: "a-20260518-001", Timestamp: ts, Description: "First"},
			{ID: "a-20260518-002", Timestamp: ts, Description: "Second"},
		},
	}
	if err := s.SaveActivityLog(al); err != nil {
		t.Fatal(err)
	}

	refs := []model.ActivityRef{
		{ID: "a-20260518-001", Date: "2026-05-18"},
	}
	results, err := s.GetActivitiesByRefs(refs, date)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Description != "First" {
		t.Fatalf("unexpected results: %v", results)
	}
}

func TestGetActivitiesByRefsFallback(t *testing.T) {
	s := newTestStore(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	ts := time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC)

	al := &model.ActivityLog{
		Date: date,
		Entries: []model.ActivityEntry{
			{ID: "a-20260518-001", Timestamp: ts, Description: "Only entry"},
		},
	}
	if err := s.SaveActivityLog(al); err != nil {
		t.Fatal(err)
	}

	// Empty refs triggers fallback: load all entries for fallbackDate
	results, err := s.GetActivitiesByRefs(nil, date)
	if err != nil {
		t.Fatal(err)
	}
	if len(results) != 1 || results[0].Description != "Only entry" {
		t.Fatalf("unexpected results: %v", results)
	}
}

func TestAddActivityRefToTask(t *testing.T) {
	s := newTestStore(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	plan := &model.DayPlan{
		Date: date,
		Tasks: []model.Task{
			{ID: "t-20260518-001", Title: "Task", Status: model.StatusTodo, CreatedAt: date, UpdatedAt: date},
		},
	}
	if err := s.SaveDayPlan(plan); err != nil {
		t.Fatal(err)
	}

	ref := model.ActivityRef{ID: "a-20260518-001", Date: "2026-05-18"}
	if err := s.AddActivityRefToTask("t-20260518-001", ref); err != nil {
		t.Fatalf("AddActivityRefToTask: %v", err)
	}

	task, _, err := s.GetTaskByID("t-20260518-001")
	if err != nil {
		t.Fatal(err)
	}
	if len(task.ActivityRefs) != 1 || task.ActivityRefs[0].ID != "a-20260518-001" {
		t.Errorf("expected activity ref, got %v", task.ActivityRefs)
	}

	// Adding the same ref again should be idempotent
	if err := s.AddActivityRefToTask("t-20260518-001", ref); err != nil {
		t.Fatal(err)
	}
	task, _, err = s.GetTaskByID("t-20260518-001")
	if err != nil {
		t.Fatal(err)
	}
	if len(task.ActivityRefs) != 1 {
		t.Errorf("expected 1 ref (idempotent), got %d", len(task.ActivityRefs))
	}
}

func TestAddActivityRefToTaskNotFound(t *testing.T) {
	s := newTestStore(t)
	ref := model.ActivityRef{ID: "a-20260518-001", Date: "2026-05-18"}
	err := s.AddActivityRefToTask("t-20260518-999", ref)
	if err == nil {
		t.Error("expected error for missing task")
	}
}

func TestGetTaskByIDNotFound(t *testing.T) {
	s := newTestStore(t)
	_, _, err := s.GetTaskByID("t-20260101-999")
	if err == nil {
		t.Error("expected error for missing task")
	}
}

// ── GetActivityLog corrupt YAML ───────────────────────────────────────────────

func TestGetActivityLogCorruptYAML(t *testing.T) {
	s := newTestStore(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)

	// Write invalid YAML to the activity file path
	path := fmt.Sprintf("%s/activity/2026-05-18.yaml", s.dataDir)
	if err := os.WriteFile(path, []byte("{{invalid yaml"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := s.GetActivityLog(date)
	if err == nil {
		t.Error("expected error for corrupt activity YAML")
	}
}

// ── GetDayPlan corrupt YAML ───────────────────────────────────────────────────

func TestGetDayPlanCorruptYAML(t *testing.T) {
	s := newTestStore(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)

	// Write invalid YAML to the day file path
	path := fmt.Sprintf("%s/days/2026-05-18.yaml", s.dataDir)
	if err := os.WriteFile(path, []byte("{{invalid yaml"), 0644); err != nil {
		t.Fatalf("WriteFile: %v", err)
	}

	_, err := s.GetDayPlan(date)
	if err == nil {
		t.Error("expected error for corrupt day YAML")
	}
}

// ── GetActivitiesByRefs with invalid date in ref ──────────────────────────────

func TestGetActivitiesByRefsInvalidDate(t *testing.T) {
	s := newTestStore(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)

	refs := []model.ActivityRef{
		{ID: "a-20260518-001", Date: "not-a-date"},
	}
	results, err := s.GetActivitiesByRefs(refs, date)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 0 {
		t.Errorf("expected 0 results for invalid date ref, got %d", len(results))
	}
}

// ── SaveDayPlan write error ───────────────────────────────────────────────────

func TestSaveDayPlanWriteError(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewYAMLStore(dir)

	daysDir := filepath.Join(dir, "days")
	if err := os.Chmod(daysDir, 0555); err != nil {
		t.Skip("cannot chmod:", err)
	}
	defer os.Chmod(daysDir, 0755)

	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	plan := &model.DayPlan{Date: date, Tasks: []model.Task{}}
	if err := s.SaveDayPlan(plan); err == nil {
		t.Error("expected error writing to read-only directory")
	}
}

// ── SaveActivityLog write error ───────────────────────────────────────────────

func TestSaveActivityLogWriteError(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewYAMLStore(dir)

	actDir := filepath.Join(dir, "activity")
	if err := os.Chmod(actDir, 0555); err != nil {
		t.Skip("cannot chmod:", err)
	}
	defer os.Chmod(actDir, 0755)

	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	al := &model.ActivityLog{Date: date, Entries: []model.ActivityEntry{}}
	if err := s.SaveActivityLog(al); err == nil {
		t.Error("expected error writing to read-only directory")
	}
}

// ── ListTasks / ListActivity skip non-yaml and bad-date files ─────────────────

func TestListTasksSkipsNonYAMLFiles(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewYAMLStore(dir)

	// Drop a non-yaml file and a yaml with invalid date name into the days dir
	os.WriteFile(filepath.Join(dir, "days", "README.txt"), []byte("not yaml"), 0644)
	os.WriteFile(filepath.Join(dir, "days", "not-a-date.yaml"), []byte("{}"), 0644)

	results, err := s.ListTasks(TaskFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = results // just must not crash
}

func TestListActivitySkipsNonYAMLFiles(t *testing.T) {
	dir := t.TempDir()
	s, _ := NewYAMLStore(dir)

	os.WriteFile(filepath.Join(dir, "activity", "README.txt"), []byte("not yaml"), 0644)
	os.WriteFile(filepath.Join(dir, "activity", "not-a-date.yaml"), []byte("{}"), 0644)

	results, err := s.ListActivity(ActivityFilter{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	_ = results
}

// ── GetActivitiesByRefs sort path ─────────────────────────────────────────────

func TestGetActivitiesByRefsSortsByTimestamp(t *testing.T) {
	s := newTestStore(t)
	d1 := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2026, 5, 19, 0, 0, 0, 0, time.UTC)
	ts1 := time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC)
	ts2 := time.Date(2026, 5, 19, 8, 0, 0, 0, time.UTC)

	s.SaveActivityLog(&model.ActivityLog{Date: d1, Entries: []model.ActivityEntry{
		{ID: "a-20260518-001", Timestamp: ts1, Description: "First"},
	}})
	s.SaveActivityLog(&model.ActivityLog{Date: d2, Entries: []model.ActivityEntry{
		{ID: "a-20260519-001", Timestamp: ts2, Description: "Second"},
	}})

	refs := []model.ActivityRef{
		{ID: "a-20260519-001", Date: "2026-05-19"},
		{ID: "a-20260518-001", Date: "2026-05-18"},
	}
	results, err := s.GetActivitiesByRefs(refs, d1)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(results) != 2 {
		t.Fatalf("expected 2 results, got %d", len(results))
	}
	if !results[0].Timestamp.Before(results[1].Timestamp) {
		t.Error("results should be sorted by timestamp ascending")
	}
}

func TestContainsAll(t *testing.T) {
	cases := []struct {
		haystack []string
		needles  []string
		want     bool
	}{
		{[]string{"a", "b", "c"}, []string{"a", "c"}, true},
		{[]string{"a", "b"}, []string{"a", "c"}, false},
		{[]string{"a"}, []string{}, true},
		{[]string{}, []string{"a"}, false},
		{[]string{"x"}, []string{"x"}, true},
	}
	for _, tc := range cases {
		got := containsAll(tc.haystack, tc.needles)
		if got != tc.want {
			t.Errorf("containsAll(%v, %v) = %v, want %v", tc.haystack, tc.needles, got, tc.want)
		}
	}
}

func TestWalkDaysDateFilter(t *testing.T) {
	s := newTestStore(t)
	d1 := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC)
	d3 := time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC)
	for _, d := range []time.Time{d1, d2, d3} {
		s.SaveDayPlan(&model.DayPlan{Date: d, Tasks: []model.Task{}})
	}

	var visited []time.Time
	from, to := d1.AddDate(0, 0, 1), d3.AddDate(0, 0, -1)
	if err := s.walkDays(&from, &to, func(d time.Time) error {
		visited = append(visited, d)
		return nil
	}); err != nil {
		t.Fatalf("walkDays: %v", err)
	}

	if len(visited) != 1 || !visited[0].Equal(d2) {
		t.Errorf("expected [%v], got %v", d2, visited)
	}
}

func TestWalkActivityFilesDateFilter(t *testing.T) {
	s := newTestStore(t)
	d1 := time.Date(2026, 5, 1, 0, 0, 0, 0, time.UTC)
	d2 := time.Date(2026, 5, 15, 0, 0, 0, 0, time.UTC)
	d3 := time.Date(2026, 5, 31, 0, 0, 0, 0, time.UTC)
	for _, d := range []time.Time{d1, d2, d3} {
		s.SaveActivityLog(&model.ActivityLog{Date: d, Entries: []model.ActivityEntry{}})
	}

	var visited []time.Time
	from, to := d1.AddDate(0, 0, 1), d3.AddDate(0, 0, -1)
	if err := s.walkActivityFiles(&from, &to, func(d time.Time) error {
		visited = append(visited, d)
		return nil
	}); err != nil {
		t.Fatalf("walkActivityFiles: %v", err)
	}

	if len(visited) != 1 || !visited[0].Equal(d2) {
		t.Errorf("expected [%v], got %v", d2, visited)
	}
}
