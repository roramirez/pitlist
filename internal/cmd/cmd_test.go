package cmd

import (
	"testing"
	"time"

	"github.com/roramirez/pitlist/internal/config"
	"github.com/roramirez/pitlist/internal/model"
	"github.com/roramirez/pitlist/internal/storage"
)

// setupTest sets the package-level store (and optionally cfg) used by all commands.
func setupTest(t *testing.T) *storage.YAMLStore {
	t.Helper()
	s, err := storage.NewYAMLStore(t.TempDir())
	if err != nil {
		t.Fatalf("NewYAMLStore: %v", err)
	}
	store = s
	cfg = &config.Config{DataDir: t.TempDir()}
	return s
}

func seedTask(t *testing.T, s *storage.YAMLStore, date time.Time, task model.Task) {
	t.Helper()
	plan, _ := s.GetDayPlan(date)
	plan.Tasks = append(plan.Tasks, task)
	if err := s.SaveDayPlan(plan); err != nil {
		t.Fatalf("SaveDayPlan: %v", err)
	}
}

// ── util ──────────────────────────────────────────────────────────────────────

func TestToday(t *testing.T) {
	d := today()
	if d.Hour() != 0 || d.Minute() != 0 || d.Second() != 0 || d.Nanosecond() != 0 {
		t.Errorf("today() should have zeroed time, got %v", d)
	}
}

func TestWeekStart(t *testing.T) {
	cases := []struct {
		in   time.Time
		want time.Weekday
	}{
		{time.Date(2026, 5, 20, 0, 0, 0, 0, time.UTC), time.Monday}, // Wednesday → Monday
		{time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC), time.Monday}, // Monday → itself
		{time.Date(2026, 5, 24, 0, 0, 0, 0, time.UTC), time.Monday}, // Sunday → previous Monday
	}
	for _, c := range cases {
		got := weekStart(c.in)
		if got.Weekday() != c.want {
			t.Errorf("weekStart(%v).Weekday() = %v, want %v", c.in, got.Weekday(), c.want)
		}
	}
}

// ── add ───────────────────────────────────────────────────────────────────────

func TestAddCmd(t *testing.T) {
	s := setupTest(t)
	date := today()

	cmd := newAddCmd()
	cmd.SetArgs([]string{"My new task"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("add: %v", err)
	}

	plan, _ := s.GetDayPlan(date)
	if len(plan.Tasks) != 1 || plan.Tasks[0].Title != "My new task" {
		t.Errorf("task not added, got %v", plan.Tasks)
	}
}

func TestAddCmdWithFlags(t *testing.T) {
	s := setupTest(t)
	date := today()

	cmd := newAddCmd()
	cmd.SetArgs([]string{"Flagged task", "--priority", "high", "--label", "work", "--due", "2026-12-31"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("add with flags: %v", err)
	}

	plan, _ := s.GetDayPlan(date)
	if len(plan.Tasks) != 1 {
		t.Fatalf("expected 1 task, got %d", len(plan.Tasks))
	}
	task := plan.Tasks[0]
	if task.Priority != model.PriorityHigh {
		t.Errorf("priority = %q, want high", task.Priority)
	}
	if len(task.Labels) == 0 || task.Labels[0] != "work" {
		t.Errorf("labels = %v, want [work]", task.Labels)
	}
	if task.DueDate != "2026-12-31" {
		t.Errorf("due = %q, want 2026-12-31", task.DueDate)
	}
}

func TestAddCmdWithDate(t *testing.T) {
	s := setupTest(t)
	targetDate := time.Date(2026, 6, 1, 0, 0, 0, 0, time.UTC)

	cmd := newAddCmd()
	cmd.SetArgs([]string{"Future task", "--date", "2026-06-01"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("add --date: %v", err)
	}

	plan, _ := s.GetDayPlan(targetDate)
	if len(plan.Tasks) != 1 || plan.Tasks[0].Title != "Future task" {
		t.Errorf("task not saved on correct date, got %v", plan.Tasks)
	}
}

func TestAddCmdInvalidDate(t *testing.T) {
	setupTest(t)
	cmd := newAddCmd()
	cmd.SetArgs([]string{"Task", "--date", "not-a-date"})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error for invalid date")
	}
}

func TestAddCmdUnknownPriorityDefaultsMedium(t *testing.T) {
	s := setupTest(t)
	cmd := newAddCmd()
	cmd.SetArgs([]string{"Task", "--priority", "bogus"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("add: %v", err)
	}
	plan, _ := s.GetDayPlan(today())
	if plan.Tasks[0].Priority != model.PriorityMedium {
		t.Errorf("expected medium priority, got %q", plan.Tasks[0].Priority)
	}
}

// ── done ─────────────────────────────────────────────────────────────────────

func TestDoneCmd(t *testing.T) {
	s := setupTest(t)
	date := today()
	seedTask(t, s, date, model.Task{
		ID: "t-20260518-001", Title: "Task", Status: model.StatusTodo,
		CreatedAt: date, UpdatedAt: date,
	})

	cmd := newDoneCmd()
	cmd.SetArgs([]string{"t-20260518-001"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("done: %v", err)
	}

	plan, _ := s.GetDayPlan(date)
	if plan.Tasks[0].Status != model.StatusDone {
		t.Errorf("expected done, got %q", plan.Tasks[0].Status)
	}
	if plan.Tasks[0].DoneAt == nil {
		t.Error("DoneAt should be set")
	}
}

func TestDoneCmdNotFound(t *testing.T) {
	setupTest(t)
	cmd := newDoneCmd()
	cmd.SetArgs([]string{"t-20260518-999"})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error for missing task")
	}
}

// ── list ─────────────────────────────────────────────────────────────────────

func TestListCmdToday(t *testing.T) {
	s := setupTest(t)
	date := today()
	seedTask(t, s, date, model.Task{
		ID: "t-20260518-001", Title: "Today task", Status: model.StatusTodo,
		CreatedAt: date, UpdatedAt: date,
	})

	cmd := newListCmd()
	if err := cmd.Execute(); err != nil {
		t.Fatalf("list: %v", err)
	}
}

func TestListCmdWithLabel(t *testing.T) {
	s := setupTest(t)
	date := today()
	seedTask(t, s, date, model.Task{
		ID: "t-20260518-001", Title: "Work task", Labels: []string{"work"},
		Status: model.StatusTodo, CreatedAt: date, UpdatedAt: date,
	})

	cmd := newListCmd()
	cmd.SetArgs([]string{"--label", "work"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("list --label: %v", err)
	}
}

func TestListCmdWeek(t *testing.T) {
	setupTest(t)
	cmd := newListCmd()
	cmd.SetArgs([]string{"--week"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("list --week: %v", err)
	}
}

func TestListCmdFromTo(t *testing.T) {
	setupTest(t)
	cmd := newListCmd()
	cmd.SetArgs([]string{"--from", "2026-05-01", "--to", "2026-05-31"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("list --from --to: %v", err)
	}
}

func TestListCmdDate(t *testing.T) {
	setupTest(t)
	cmd := newListCmd()
	cmd.SetArgs([]string{"--date", "2026-05-18"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("list --date: %v", err)
	}
}

func TestListCmdInvalidFrom(t *testing.T) {
	setupTest(t)
	cmd := newListCmd()
	cmd.SetArgs([]string{"--from", "bad"})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error for invalid --from")
	}
}

func TestListCmdInvalidTo(t *testing.T) {
	setupTest(t)
	cmd := newListCmd()
	cmd.SetArgs([]string{"--to", "bad"})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error for invalid --to")
	}
}

func TestListCmdInvalidDate(t *testing.T) {
	setupTest(t)
	cmd := newListCmd()
	cmd.SetArgs([]string{"--date", "bad"})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error for invalid --date")
	}
}

// ── show ─────────────────────────────────────────────────────────────────────

func TestShowCmd(t *testing.T) {
	s := setupTest(t)
	date := today()
	seedTask(t, s, date, model.Task{
		ID: "t-20260518-001", Title: "Show me", Status: model.StatusTodo,
		Labels: []string{"work"}, DueDate: "2026-12-31", Notes: "some notes",
		CarryFrom: "2026-05-17", CreatedAt: date, UpdatedAt: date,
	})

	cmd := newShowCmd()
	cmd.SetArgs([]string{"t-20260518-001"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("show: %v", err)
	}
}

func TestShowCmdNotFound(t *testing.T) {
	setupTest(t)
	cmd := newShowCmd()
	cmd.SetArgs([]string{"t-20260518-999"})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error for missing task")
	}
}

// ── delete ────────────────────────────────────────────────────────────────────

func TestDeleteCmdForce(t *testing.T) {
	s := setupTest(t)
	date := today()
	seedTask(t, s, date, model.Task{
		ID: "t-20260518-001", Title: "Delete me", Status: model.StatusTodo,
		CreatedAt: date, UpdatedAt: date,
	})

	cmd := newDeleteCmd()
	cmd.SetArgs([]string{"t-20260518-001", "--force"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("delete --force: %v", err)
	}

	plan, _ := s.GetDayPlan(date)
	if len(plan.Tasks) != 0 {
		t.Errorf("expected 0 tasks after delete, got %d", len(plan.Tasks))
	}
}

func TestDeleteCmdNotFound(t *testing.T) {
	setupTest(t)
	cmd := newDeleteCmd()
	cmd.SetArgs([]string{"t-20260518-999", "--force"})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error for missing task")
	}
}

// ── carry ─────────────────────────────────────────────────────────────────────

func TestCarryTask(t *testing.T) {
	s := setupTest(t)
	srcDate := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	destDate := srcDate.AddDate(0, 0, 1)

	task := &model.Task{
		ID: "t-20260518-001", Title: "Carry me", Status: model.StatusTodo,
		CreatedAt: srcDate, UpdatedAt: srcDate,
	}
	seedTask(t, s, srcDate, *task)

	if err := carryTask(s, task, srcDate, destDate); err != nil {
		t.Fatalf("carryTask: %v", err)
	}

	// Removed from source
	srcPlan, _ := s.GetDayPlan(srcDate)
	if len(srcPlan.Tasks) != 0 {
		t.Errorf("expected 0 tasks on src day, got %d", len(srcPlan.Tasks))
	}

	// Added to dest
	destPlan, _ := s.GetDayPlan(destDate)
	if len(destPlan.Tasks) != 1 || destPlan.Tasks[0].ID != "t-20260518-001" {
		t.Errorf("task not found on dest day, got %v", destPlan.Tasks)
	}

	// Activity log entry created
	actLog, _ := s.GetActivityLog(srcDate)
	if len(actLog.Entries) != 1 {
		t.Errorf("expected 1 activity entry, got %d", len(actLog.Entries))
	}
}

func TestCarryCmdToTomorrow(t *testing.T) {
	s := setupTest(t)
	srcDate := today()
	task := model.Task{
		ID: "t-carry-001", Title: "Carry cmd task", Status: model.StatusTodo,
		CreatedAt: srcDate, UpdatedAt: srcDate,
	}
	seedTask(t, s, srcDate, task)

	cmd := newCarryCmd()
	cmd.SetArgs([]string{"t-carry-001"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("carry: %v", err)
	}

	srcPlan, _ := s.GetDayPlan(srcDate)
	if len(srcPlan.Tasks) != 0 {
		t.Errorf("expected task removed from src, got %d tasks", len(srcPlan.Tasks))
	}
}

func TestCarryCmdToDate(t *testing.T) {
	s := setupTest(t)
	srcDate := today()
	task := model.Task{
		ID: "t-carry-002", Title: "Carry to date", Status: model.StatusTodo,
		CreatedAt: srcDate, UpdatedAt: srcDate,
	}
	seedTask(t, s, srcDate, task)

	cmd := newCarryCmd()
	cmd.SetArgs([]string{"t-carry-002", "--to", "2026-12-01"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("carry --to: %v", err)
	}
}

func TestCarryCmdInvalidDate(t *testing.T) {
	s := setupTest(t)
	date := today()
	seedTask(t, s, date, model.Task{
		ID: "t-carry-003", Title: "Task", Status: model.StatusTodo,
		CreatedAt: date, UpdatedAt: date,
	})
	cmd := newCarryCmd()
	cmd.SetArgs([]string{"t-carry-003", "--to", "bad-date"})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error for invalid date")
	}
}

// ── log ───────────────────────────────────────────────────────────────────────

func TestLogAddCmd(t *testing.T) {
	s := setupTest(t)
	date := today()

	cmd := newLogAddCmd()
	cmd.SetArgs([]string{"Did some work"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("log add: %v", err)
	}

	actLog, _ := s.GetActivityLog(date)
	if len(actLog.Entries) != 1 || actLog.Entries[0].Description != "Did some work" {
		t.Errorf("activity not logged, got %v", actLog.Entries)
	}
}

func TestLogAddCmdWithFlags(t *testing.T) {
	s := setupTest(t)
	date := today()
	task := model.Task{
		ID: "t-log-001", Title: "Ref task", Status: model.StatusTodo,
		CreatedAt: date, UpdatedAt: date,
	}
	seedTask(t, s, date, task)

	cmd := newLogAddCmd()
	cmd.SetArgs([]string{"Worked on task", "--tag", "work", "--duration", "30", "--ref", "t-log-001"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("log add with flags: %v", err)
	}

	actLog, _ := s.GetActivityLog(date)
	if len(actLog.Entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(actLog.Entries))
	}
	e := actLog.Entries[0]
	if e.DurationMin != 30 {
		t.Errorf("duration = %d, want 30", e.DurationMin)
	}
	if e.TaskRef != "t-log-001" {
		t.Errorf("taskRef = %q, want t-log-001", e.TaskRef)
	}
}

func TestLogAddCmdWithDate(t *testing.T) {
	s := setupTest(t)
	targetDate := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)

	cmd := newLogAddCmd()
	cmd.SetArgs([]string{"Past work", "--date", "2026-05-10"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("log add --date: %v", err)
	}

	actLog, _ := s.GetActivityLog(targetDate)
	if len(actLog.Entries) != 1 {
		t.Errorf("expected 1 entry on 2026-05-10, got %d", len(actLog.Entries))
	}
}

func TestLogAddCmdInvalidDate(t *testing.T) {
	setupTest(t)
	cmd := newLogAddCmd()
	cmd.SetArgs([]string{"Work", "--date", "bad"})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error for invalid date")
	}
}

func TestLogListCmdToday(t *testing.T) {
	s := setupTest(t)
	date := today()
	al := &model.ActivityLog{
		Date: date,
		Entries: []model.ActivityEntry{
			{ID: "a-001", Timestamp: date, Description: "Entry"},
		},
	}
	s.SaveActivityLog(al)

	cmd := newLogListCmd()
	if err := cmd.Execute(); err != nil {
		t.Fatalf("log list: %v", err)
	}
}

func TestLogListCmdWeek(t *testing.T) {
	setupTest(t)
	cmd := newLogListCmd()
	cmd.SetArgs([]string{"--week"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("log list --week: %v", err)
	}
}

func TestLogListCmdFromTo(t *testing.T) {
	setupTest(t)
	cmd := newLogListCmd()
	cmd.SetArgs([]string{"--from", "2026-05-01", "--to", "2026-05-31"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("log list --from --to: %v", err)
	}
}

func TestLogListCmdDate(t *testing.T) {
	setupTest(t)
	cmd := newLogListCmd()
	cmd.SetArgs([]string{"--date", "2026-05-18"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("log list --date: %v", err)
	}
}

func TestLogListCmdInvalidFrom(t *testing.T) {
	setupTest(t)
	cmd := newLogListCmd()
	cmd.SetArgs([]string{"--from", "bad"})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error for invalid --from")
	}
}

func TestLogListCmdInvalidTo(t *testing.T) {
	setupTest(t)
	cmd := newLogListCmd()
	cmd.SetArgs([]string{"--to", "bad"})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error for invalid --to")
	}
}

func TestLogListCmdInvalidDate(t *testing.T) {
	setupTest(t)
	cmd := newLogListCmd()
	cmd.SetArgs([]string{"--date", "bad"})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error for invalid --date")
	}
}

func TestLogLinkCmd(t *testing.T) {
	s := setupTest(t)
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)

	seedTask(t, s, date, model.Task{
		ID: "t-20260518-001", Title: "Task", Status: model.StatusTodo,
		CreatedAt: date, UpdatedAt: date,
	})

	al := &model.ActivityLog{
		Date: date,
		Entries: []model.ActivityEntry{
			{ID: "a-20260518-001", Timestamp: date, Description: "Work"},
		},
	}
	s.SaveActivityLog(al)

	cmd := newLogLinkCmd()
	cmd.SetArgs([]string{"a-20260518-001", "t-20260518-001"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("log link: %v", err)
	}

	updated, _ := s.GetActivityLog(date)
	if updated.Entries[0].TaskRef != "t-20260518-001" {
		t.Errorf("TaskRef = %q, want t-20260518-001", updated.Entries[0].TaskRef)
	}
}

func TestLogLinkCmdInvalidActivityID(t *testing.T) {
	s := setupTest(t)
	date := today()
	seedTask(t, s, date, model.Task{
		ID: "t-link-001", Title: "T", Status: model.StatusTodo,
		CreatedAt: date, UpdatedAt: date,
	})
	cmd := newLogLinkCmd()
	cmd.SetArgs([]string{"short", "t-link-001"})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error for short activity ID")
	}
}

func TestLogLinkCmdActivityNotFound(t *testing.T) {
	s := setupTest(t)
	date := today()
	seedTask(t, s, date, model.Task{
		ID: "t-link-002", Title: "T", Status: model.StatusTodo,
		CreatedAt: date, UpdatedAt: date,
	})
	cmd := newLogLinkCmd()
	cmd.SetArgs([]string{"a-20260518-999", "t-link-002"})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error for missing activity")
	}
}

func TestLogLinkCmdTaskNotFound(t *testing.T) {
	setupTest(t)
	cmd := newLogLinkCmd()
	cmd.SetArgs([]string{"a-20260518-001", "t-99999999-999"})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error for missing task")
	}
}

// ── agenda ────────────────────────────────────────────────────────────────────

func TestAgendaCmd(t *testing.T) {
	setupTest(t)
	cmd := newAgendaCmd()
	if err := cmd.Execute(); err != nil {
		t.Fatalf("agenda: %v", err)
	}
}

func TestAgendaCmdWithTasks(t *testing.T) {
	s := setupTest(t)
	date := today()
	seedTask(t, s, date, model.Task{
		ID: "t-agenda-001", Title: "Pending task", Labels: []string{"work"},
		Status: model.StatusTodo, CreatedAt: date, UpdatedAt: date,
	})

	cmd := newAgendaCmd()
	cmd.SetArgs([]string{"--label", "work"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("agenda --label: %v", err)
	}
}

func TestAgendaCmdFromTo(t *testing.T) {
	setupTest(t)
	cmd := newAgendaCmd()
	cmd.SetArgs([]string{"--from", "2026-05-01", "--to", "2026-05-31"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("agenda --from --to: %v", err)
	}
}

func TestAgendaCmdFromOnly(t *testing.T) {
	setupTest(t)
	cmd := newAgendaCmd()
	cmd.SetArgs([]string{"--from", "2026-05-01"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("agenda --from: %v", err)
	}
}

func TestAgendaCmdInvalidFrom(t *testing.T) {
	setupTest(t)
	cmd := newAgendaCmd()
	cmd.SetArgs([]string{"--from", "bad"})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error for invalid --from")
	}
}

func TestAgendaCmdInvalidTo(t *testing.T) {
	setupTest(t)
	cmd := newAgendaCmd()
	cmd.SetArgs([]string{"--from", "2026-05-01", "--to", "bad"})
	if err := cmd.Execute(); err == nil {
		t.Error("expected error for invalid --to")
	}
}

// ── stats ─────────────────────────────────────────────────────────────────────

func TestStatsCmd(t *testing.T) {
	setupTest(t)
	cmd := newStatsCmd()
	if err := cmd.Execute(); err != nil {
		t.Fatalf("stats: %v", err)
	}
}

func TestStatsCmdWeek(t *testing.T) {
	setupTest(t)
	cmd := newStatsCmd()
	cmd.SetArgs([]string{"--week"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("stats --week: %v", err)
	}
}

func TestStatsCmdMonth(t *testing.T) {
	setupTest(t)
	cmd := newStatsCmd()
	cmd.SetArgs([]string{"--month"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("stats --month: %v", err)
	}
}

func TestStatsCmdWithData(t *testing.T) {
	s := setupTest(t)
	date := today()
	seedTask(t, s, date, model.Task{
		ID: "t-stats-001", Title: "Done", Status: model.StatusDone,
		CreatedAt: date, UpdatedAt: date,
	})
	al := &model.ActivityLog{
		Date: date,
		Entries: []model.ActivityEntry{
			{ID: "a-stats-001", Timestamp: date, Description: "Work", DurationMin: 90},
		},
	}
	s.SaveActivityLog(al)

	cmd := newStatsCmd()
	if err := cmd.Execute(); err != nil {
		t.Fatalf("stats with data: %v", err)
	}
}

// ── log (parent cmd wrapper) ──────────────────────────────────────────────────

func TestLogCmdNoArgs(t *testing.T) {
	setupTest(t)
	cmd := newLogCmd()
	// No args → prints help (no error)
	if err := cmd.Execute(); err != nil {
		t.Fatalf("log with no args: %v", err)
	}
}

func TestLogCmdWithDescription(t *testing.T) {
	s := setupTest(t)
	date := today()

	cmd := newLogCmd()
	cmd.SetArgs([]string{"Work via parent cmd"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("log <desc>: %v", err)
	}

	actLog, _ := s.GetActivityLog(date)
	if len(actLog.Entries) != 1 {
		t.Errorf("expected 1 entry, got %d", len(actLog.Entries))
	}
}

// ── pure helpers ─────────────────────────────────────────────────────────────

func TestMatchLabels(t *testing.T) {
	if !matchLabels([]string{"work", "auth"}, nil) {
		t.Error("empty filter should match all")
	}
	if !matchLabels([]string{"work", "auth"}, []string{"work"}) {
		t.Error("task with label should match filter")
	}
	if matchLabels([]string{"personal"}, []string{"work"}) {
		t.Error("task without label should not match")
	}
	if matchLabels(nil, []string{"work"}) {
		t.Error("task with no labels should not match non-empty filter")
	}
}

func TestPrintTask(t *testing.T) {
	date := time.Date(2026, 5, 18, 0, 0, 0, 0, time.UTC)
	cases := []model.Task{
		{ID: "t-001", Title: "Todo", Status: model.StatusTodo},
		{ID: "t-002", Title: "Done", Status: model.StatusDone},
		{ID: "t-003", Title: "In progress", Status: model.StatusInProgress},
		{ID: "t-004", Title: "Cancelled", Status: model.StatusCancelled},
		{ID: "t-005", Title: "High priority", Status: model.StatusTodo, Priority: model.PriorityHigh},
		{ID: "t-006", Title: "With label", Status: model.StatusTodo, Labels: []string{"work"}},
		{ID: "t-007", Title: "Carried", Status: model.StatusTodo, CarryFrom: date.Format("2006-01-02")},
	}
	for _, task := range cases {
		tc := task // no panic is good enough
		printTask(&tc)
	}
}

func TestPrintActivityEntry(t *testing.T) {
	date := time.Date(2026, 5, 18, 10, 0, 0, 0, time.UTC)
	cases := []model.ActivityEntry{
		{ID: "a-001", Timestamp: date, Description: "Basic"},
		{ID: "a-002", Timestamp: date, Description: "With duration", DurationMin: 45},
		{ID: "a-003", Timestamp: date, Description: "With tags", Tags: []string{"work"}},
		{ID: "a-004", Timestamp: date, Description: "With ref", TaskRef: "t-001"},
	}
	for _, e := range cases {
		ec := e
		printActivityEntry(&ec)
	}
}

func TestPrintDayHeader(t *testing.T) {
	t0 := today()
	printDayHeader(t0)                   // today
	printDayHeader(t0.AddDate(0, 0, 1))  // tomorrow
	printDayHeader(t0.AddDate(0, 0, -1)) // overdue
	printDayHeader(t0.AddDate(0, 0, 3))  // future (no tag)
}
