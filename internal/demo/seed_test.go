package demo

import (
	"os"
	"testing"
	"time"

	"github.com/roramirez/pitlist/internal/storage"
)

func todayUTC() time.Time {
	now := time.Now()
	return time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
}

func newTestStore(t *testing.T) *storage.YAMLStore {
	t.Helper()
	s, err := storage.NewYAMLStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewYAMLStore: %v", err)
	}
	return s
}

func TestSeedIntoTodayTasks(t *testing.T) {
	s := newTestStore(t)
	if err := SeedInto(s); err != nil {
		t.Fatalf("SeedInto: %v", err)
	}

	today := todayUTC()
	plan, err := s.GetDayPlan(today)
	if err != nil {
		t.Fatalf("GetDayPlan(today): %v", err)
	}
	if len(plan.Tasks) < 8 {
		t.Errorf("today plan: want >=8 tasks, got %d", len(plan.Tasks))
	}

	// verify work and personal contexts both present
	var hasWork, hasPersonal bool
	for _, task := range plan.Tasks {
		if task.Context == "work" {
			hasWork = true
		}
		if task.Context == "personal" {
			hasPersonal = true
		}
	}
	if !hasWork {
		t.Error("no work tasks seeded for today")
	}
	if !hasPersonal {
		t.Error("no personal tasks seeded for today")
	}
}

func TestSeedIntoYesterdayTasks(t *testing.T) {
	s := newTestStore(t)
	if err := SeedInto(s); err != nil {
		t.Fatalf("SeedInto: %v", err)
	}

	yesterday := todayUTC().AddDate(0, 0, -1)
	plan, err := s.GetDayPlan(yesterday)
	if err != nil {
		t.Fatalf("GetDayPlan(yesterday): %v", err)
	}
	if len(plan.Tasks) < 3 {
		t.Errorf("yesterday plan: want >=3 tasks, got %d", len(plan.Tasks))
	}
}

func TestSeedIntoTodayActivity(t *testing.T) {
	s := newTestStore(t)
	if err := SeedInto(s); err != nil {
		t.Fatalf("SeedInto: %v", err)
	}

	today := todayUTC()
	log, err := s.GetActivityLog(today)
	if err != nil {
		t.Fatalf("GetActivityLog(today): %v", err)
	}
	if len(log.Entries) < 4 {
		t.Errorf("today activity: want >=4 entries, got %d", len(log.Entries))
	}

	// at least one personal activity (fitness) and one work activity
	var hasPersonal, hasWork bool
	for _, e := range log.Entries {
		for _, tag := range e.Tags {
			if tag == "fitness" || tag == "learning" {
				hasPersonal = true
			}
			if tag == "ci" || tag == "review" || tag == "meeting" {
				hasWork = true
			}
		}
	}
	if !hasPersonal {
		t.Error("no personal activity entry seeded for today")
	}
	if !hasWork {
		t.Error("no work activity entry seeded for today")
	}
}

func TestSeedIntoYesterdayActivity(t *testing.T) {
	s := newTestStore(t)
	if err := SeedInto(s); err != nil {
		t.Fatalf("SeedInto: %v", err)
	}

	yesterday := todayUTC().AddDate(0, 0, -1)
	log, err := s.GetActivityLog(yesterday)
	if err != nil {
		t.Fatalf("GetActivityLog(yesterday): %v", err)
	}
	if len(log.Entries) < 3 {
		t.Errorf("yesterday activity: want >=3 entries, got %d", len(log.Entries))
	}
}

func TestSeedIntoActivityLinkedToTask(t *testing.T) {
	s := newTestStore(t)
	if err := SeedInto(s); err != nil {
		t.Fatalf("SeedInto: %v", err)
	}

	today := todayUTC()
	plan, _ := s.GetDayPlan(today)
	var linkedTask bool
	for _, task := range plan.Tasks {
		if len(task.ActivityRefs) > 0 {
			linkedTask = true
			break
		}
	}
	if !linkedTask {
		t.Error("expected at least one task with activity refs")
	}
}

func TestSeedCreatesAndCleansTemp(t *testing.T) {
	dir, cleanup, err := Seed()
	if err != nil {
		t.Fatalf("Seed: %v", err)
	}

	if _, err := os.Stat(dir); err != nil {
		t.Errorf("data dir should exist after Seed: %v", err)
	}

	cleanup()

	if _, err := os.Stat(dir); err == nil {
		t.Error("data dir should be removed after cleanup()")
	}
}

func TestSeedTempDirHasData(t *testing.T) {
	dir, cleanup, err := Seed()
	if err != nil {
		t.Fatalf("Seed: %v", err)
	}
	defer cleanup()

	daysDir := dir + "/days"
	entries, err := os.ReadDir(daysDir)
	if err != nil {
		t.Fatalf("ReadDir days: %v", err)
	}
	if len(entries) == 0 {
		t.Error("days/ dir should contain at least one YAML file")
	}
}
