package storage

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/lithammer/fuzzysearch/fuzzy"
	"github.com/roramirez/pitlist/internal/model"
	"gopkg.in/yaml.v3"
)

type YAMLStore struct {
	dataDir string
	git     *gitHelper
}

func NewYAMLStore(dataDir string) (*YAMLStore, error) {
	for _, sub := range []string{"days", "activity"} {
		if err := os.MkdirAll(filepath.Join(dataDir, sub), 0755); err != nil {
			return nil, err
		}
	}
	g := newGitHelper(dataDir)
	_ = g.init()
	return &YAMLStore{dataDir: dataDir, git: g}, nil
}

// --- Task methods ---

func (s *YAMLStore) GetDayPlan(date time.Time) (*model.DayPlan, error) {
	path := s.dayPath(date)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &model.DayPlan{Date: date, Tasks: []model.Task{}}, nil
	}
	if err != nil {
		return nil, err
	}
	var plan model.DayPlan
	if err := yaml.Unmarshal(data, &plan); err != nil {
		return nil, err
	}
	return &plan, nil
}

func (s *YAMLStore) SaveDayPlan(plan *model.DayPlan) error {
	data, err := yaml.Marshal(plan)
	if err != nil {
		return err
	}
	path := s.dayPath(plan.Date)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return err
	}
	_ = s.git.autoCommit(path, "tasks: save "+plan.Date.Format(model.DateFormat))
	return nil
}

func (s *YAMLStore) GetTaskByID(id string) (*model.Task, time.Time, error) {
	entries, err := os.ReadDir(filepath.Join(s.dataDir, "days"))
	if err != nil {
		return nil, time.Time{}, err
	}
	for _, e := range entries {
		date, ok := parseDateFromFilename(e.Name())
		if !ok {
			continue
		}
		plan, err := s.GetDayPlan(date)
		if err != nil {
			continue
		}
		for i := range plan.Tasks {
			if plan.Tasks[i].ID == id {
				t := plan.Tasks[i]
				return &t, date, nil
			}
		}
	}
	return nil, time.Time{}, fmt.Errorf("task %q not found", id)
}

func (s *YAMLStore) ListTasks(filter TaskFilter) ([]*model.Task, error) {
	entries, err := os.ReadDir(filepath.Join(s.dataDir, "days"))
	if err != nil {
		return nil, err
	}

	var results []*model.Task
	for _, e := range entries {
		date, ok := parseDateFromFilename(e.Name())
		if !ok {
			continue
		}
		if filter.From != nil && date.Before(*filter.From) {
			continue
		}
		if filter.To != nil && date.After(*filter.To) {
			continue
		}

		plan, err := s.GetDayPlan(date)
		if err != nil {
			continue
		}
		for i := range plan.Tasks {
			t := &plan.Tasks[i]
			if matchesTaskFilter(t, filter) {
				tc := *t
				results = append(results, &tc)
			}
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].CreatedAt.Before(results[j].CreatedAt)
	})
	return results, nil
}

func matchesTaskFilter(t *model.Task, f TaskFilter) bool {
	if len(f.Statuses) > 0 {
		matched := false
		for _, s := range f.Statuses {
			if t.Status == s {
				matched = true
				break
			}
		}
		if !matched {
			return false
		}
	}
	if len(f.Labels) > 0 {
		for _, want := range f.Labels {
			found := false
			for _, l := range t.Labels {
				if l == want {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}
	if f.Search != "" && !fuzzy.MatchFold(f.Search, t.Title) {
		return false
	}
	return true
}

// --- Activity methods ---

func (s *YAMLStore) GetActivityLog(date time.Time) (*model.ActivityLog, error) {
	path := s.activityPath(date)
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return &model.ActivityLog{Date: date, Entries: []model.ActivityEntry{}}, nil
	}
	if err != nil {
		return nil, err
	}
	var log model.ActivityLog
	if err := yaml.Unmarshal(data, &log); err != nil {
		return nil, err
	}
	return &log, nil
}

func (s *YAMLStore) SaveActivityLog(log *model.ActivityLog) error {
	data, err := yaml.Marshal(log)
	if err != nil {
		return err
	}
	path := s.activityPath(log.Date)
	if err := os.WriteFile(path, data, 0644); err != nil {
		return err
	}
	_ = s.git.autoCommit(path, "activity: save "+log.Date.Format(model.DateFormat))
	return nil
}

func (s *YAMLStore) ListActivity(filter ActivityFilter) ([]*model.ActivityEntry, error) {
	entries, err := os.ReadDir(filepath.Join(s.dataDir, "activity"))
	if err != nil {
		return nil, err
	}

	var results []*model.ActivityEntry
	for _, e := range entries {
		date, ok := parseDateFromFilename(e.Name())
		if !ok {
			continue
		}
		if filter.From != nil && date.Before(*filter.From) {
			continue
		}
		if filter.To != nil && date.After(*filter.To) {
			continue
		}

		log, err := s.GetActivityLog(date)
		if err != nil {
			continue
		}
		for i := range log.Entries {
			ae := &log.Entries[i]
			if matchesActivityFilter(ae, filter) {
				ec := *ae
				results = append(results, &ec)
			}
		}
	}

	sort.Slice(results, func(i, j int) bool {
		return results[i].Timestamp.Before(results[j].Timestamp)
	})
	return results, nil
}

func matchesActivityFilter(e *model.ActivityEntry, f ActivityFilter) bool {
	if f.TaskRef != "" && e.TaskRef != f.TaskRef {
		return false
	}
	if len(f.Tags) > 0 {
		for _, want := range f.Tags {
			found := false
			for _, tag := range e.Tags {
				if tag == want {
					found = true
					break
				}
			}
			if !found {
				return false
			}
		}
	}
	if f.Search != "" && !fuzzy.MatchFold(f.Search, e.Description) {
		return false
	}
	return true
}

// --- Activity ref methods ---

func (s *YAMLStore) GetActivitiesByRefs(refs []model.ActivityRef, fallbackDate time.Time) ([]*model.ActivityEntry, error) {
	if len(refs) == 0 {
		log, err := s.GetActivityLog(fallbackDate)
		if err != nil {
			return nil, err
		}
		var out []*model.ActivityEntry
		for i := range log.Entries {
			e := log.Entries[i]
			out = append(out, &e)
		}
		return out, nil
	}

	dateIDs := make(map[string]map[string]bool)
	for _, r := range refs {
		if dateIDs[r.Date] == nil {
			dateIDs[r.Date] = make(map[string]bool)
		}
		dateIDs[r.Date][r.ID] = true
	}

	var out []*model.ActivityEntry
	for dateStr, ids := range dateIDs {
		date, err := time.Parse(model.DateFormat, dateStr)
		if err != nil {
			continue
		}
		log, err := s.GetActivityLog(date)
		if err != nil {
			continue
		}
		for i := range log.Entries {
			if ids[log.Entries[i].ID] {
				e := log.Entries[i]
				out = append(out, &e)
			}
		}
	}

	sort.Slice(out, func(i, j int) bool {
		return out[i].Timestamp.Before(out[j].Timestamp)
	})
	return out, nil
}

func (s *YAMLStore) AddActivityRefToTask(taskID string, ref model.ActivityRef) error {
	_, date, err := s.GetTaskByID(taskID)
	if err != nil {
		return err
	}
	plan, err := s.GetDayPlan(date)
	if err != nil {
		return err
	}
	for i := range plan.Tasks {
		if plan.Tasks[i].ID == taskID {
			for _, existing := range plan.Tasks[i].ActivityRefs {
				if existing.ID == ref.ID {
					return nil // already present
				}
			}
			plan.Tasks[i].ActivityRefs = append(plan.Tasks[i].ActivityRefs, ref)
			plan.Tasks[i].UpdatedAt = time.Now().UTC()
			break
		}
	}
	return s.SaveDayPlan(plan)
}

// --- ID generation ---

func NextTaskID(plan *model.DayPlan) string {
	return fmt.Sprintf("t-%s-%03d", plan.Date.Format("20060102"), len(plan.Tasks)+1)
}

func NextActivityID(log *model.ActivityLog) string {
	return fmt.Sprintf("a-%s-%03d", log.Date.Format("20060102"), len(log.Entries)+1)
}

// --- Helpers ---

func parseDateFromFilename(name string) (time.Time, bool) {
	if !strings.HasSuffix(name, ".yaml") {
		return time.Time{}, false
	}
	d, err := time.Parse(model.DateFormat, strings.TrimSuffix(name, ".yaml"))
	return d, err == nil
}

func (s *YAMLStore) dayPath(date time.Time) string {
	return filepath.Join(s.dataDir, "days", date.Format(model.DateFormat)+".yaml")
}

func (s *YAMLStore) activityPath(date time.Time) string {
	return filepath.Join(s.dataDir, "activity", date.Format(model.DateFormat)+".yaml")
}
