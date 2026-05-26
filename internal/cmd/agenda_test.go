package cmd

import (
	"testing"
	"time"

	"github.com/roramirez/pitlist/internal/model"
)

func TestFilterAgendaTasks(t *testing.T) {
	now := time.Now().UTC()
	doneAt := now

	cases := []struct {
		name     string
		tasks    []model.Task
		labels   []string
		showDone bool
		wantLen  int
	}{
		{
			name: "exclude cancelled",
			tasks: []model.Task{
				{ID: "1", Status: model.StatusCancelled},
				{ID: "2", Status: model.StatusTodo},
			},
			wantLen: 1,
		},
		{
			name: "exclude done when showDone=false",
			tasks: []model.Task{
				{ID: "1", Status: model.StatusDone, DoneAt: &doneAt},
				{ID: "2", Status: model.StatusTodo},
			},
			showDone: false,
			wantLen:  1,
		},
		{
			name: "include done when showDone=true",
			tasks: []model.Task{
				{ID: "1", Status: model.StatusDone, DoneAt: &doneAt},
				{ID: "2", Status: model.StatusTodo},
			},
			showDone: true,
			wantLen:  2,
		},
		{
			name: "filter by label",
			tasks: []model.Task{
				{ID: "1", Status: model.StatusTodo, Labels: []string{"work"}},
				{ID: "2", Status: model.StatusTodo, Labels: []string{"personal"}},
			},
			labels:  []string{"work"},
			wantLen: 1,
		},
		{
			name: "no labels filter = all pass",
			tasks: []model.Task{
				{ID: "1", Status: model.StatusTodo, Labels: []string{"work"}},
				{ID: "2", Status: model.StatusTodo},
			},
			wantLen: 2,
		},
		{
			name:    "empty input",
			tasks:   []model.Task{},
			wantLen: 0,
		},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := filterAgendaTasks(c.tasks, c.labels, c.showDone)
			if len(got) != c.wantLen {
				t.Errorf("filterAgendaTasks: got %d tasks, want %d", len(got), c.wantLen)
			}
		})
	}
}

func TestMatchLabelsAgenda(t *testing.T) {
	cases := []struct {
		name         string
		taskLabels   []string
		filterLabels []string
		want         bool
	}{
		{"no filter = always match", []string{"work"}, nil, true},
		{"all required labels present", []string{"work", "auth"}, []string{"work", "auth"}, true},
		{"missing one label", []string{"work"}, []string{"work", "auth"}, false},
		{"task has no labels", []string{}, []string{"work"}, false},
		{"empty task labels, no filter", []string{}, nil, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := matchLabels(c.taskLabels, c.filterLabels)
			if got != c.want {
				t.Errorf("matchLabels(%v, %v) = %v, want %v", c.taskLabels, c.filterLabels, got, c.want)
			}
		})
	}
}

func TestResolveAgendaRange(t *testing.T) {
	cases := []struct {
		name     string
		from, to string
		days     int
		wantDiff int // end - start in days
		wantErr  bool
	}{
		{"default range", "", "", 7, 6, false},
		{"from only", "today", "", 3, 2, false},
		{"from and to", "today", "today", 0, 0, false},
		{"invalid from", "not-a-date", "", 7, 0, true},
		{"invalid to", "today", "not-a-date", 7, 0, true},
	}

	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			start, end, err := resolveAgendaRange(c.from, c.to, c.days)
			if c.wantErr {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			diff := int(end.Sub(start).Hours() / 24)
			if diff != c.wantDiff {
				t.Errorf("range diff = %d days, want %d", diff, c.wantDiff)
			}
		})
	}
}
