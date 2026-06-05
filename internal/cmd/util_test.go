package cmd

import (
	"testing"
	"time"
)

func TestParseDate(t *testing.T) {
	t0 := today()
	cases := []struct {
		input   string
		want    time.Time
		wantErr bool
	}{
		{"2026-06-15", time.Date(2026, 6, 15, 0, 0, 0, 0, time.UTC), false},
		{"not-a-date", time.Time{}, true},
		{"today", t0, false},
		{"tomorrow", t0.AddDate(0, 0, 1), false},
		{"yesterday", t0.AddDate(0, 0, -1), false},
		{"next_week", weekStart(t0).AddDate(0, 0, 7), false},
		{"last_week", weekStart(t0).AddDate(0, 0, -7), false},
		{"in_a_week", t0.AddDate(0, 0, 7), false},
		{"next_month", time.Date(t0.Year(), t0.Month()+1, 1, 0, 0, 0, 0, t0.Location()), false},
		{"last_month", time.Date(t0.Year(), t0.Month()-1, 1, 0, 0, 0, 0, t0.Location()), false},
		{"in_a_month", t0.AddDate(0, 0, 30), false},
		{"monday", thisOrNextWeekday(t0, time.Monday), false},
		{"tuesday", thisOrNextWeekday(t0, time.Tuesday), false},
		{"wednesday", thisOrNextWeekday(t0, time.Wednesday), false},
		{"thursday", thisOrNextWeekday(t0, time.Thursday), false},
		{"friday", thisOrNextWeekday(t0, time.Friday), false},
		{"saturday", thisOrNextWeekday(t0, time.Saturday), false},
		{"sunday", thisOrNextWeekday(t0, time.Sunday), false},
		{"next_monday", strictlyNextWeekday(t0, time.Monday), false},
		{"next_tuesday", strictlyNextWeekday(t0, time.Tuesday), false},
		{"next_wednesday", strictlyNextWeekday(t0, time.Wednesday), false},
		{"next_thursday", strictlyNextWeekday(t0, time.Thursday), false},
		{"next_friday", strictlyNextWeekday(t0, time.Friday), false},
		{"next_saturday", strictlyNextWeekday(t0, time.Saturday), false},
		{"next_sunday", strictlyNextWeekday(t0, time.Sunday), false},
	}
	for _, tc := range cases {
		t.Run(tc.input, func(t *testing.T) {
			got, err := parseDate(tc.input)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !got.Equal(tc.want) {
				t.Errorf("got %v, want %v", got, tc.want)
			}
		})
	}
}

func TestParseDateNextKeywordsNeverToday(t *testing.T) {
	t0 := today()
	keywords := []string{
		"next_monday", "next_tuesday", "next_wednesday", "next_thursday",
		"next_friday", "next_saturday", "next_sunday",
	}
	for _, kw := range keywords {
		got, _ := parseDate(kw)
		if got.Equal(t0) {
			t.Errorf("%s returned today; must be strictly in the future", kw)
		}
	}
}

func TestParseDateRange(t *testing.T) {
	t0 := today()
	mon := weekStart(t0)
	fixedFrom := time.Date(2026, 6, 2, 0, 0, 0, 0, time.UTC)
	fixedTo := time.Date(2026, 6, 5, 0, 0, 0, 0, time.UTC)

	cases := []struct {
		name     string
		week     bool
		fromStr  string
		toStr    string
		dateStr  string
		wantFrom time.Time
		wantTo   time.Time
		wantErr  bool
	}{
		{
			name:     "no flags defaults to today",
			wantFrom: t0,
			wantTo:   t0,
		},
		{
			name:     "only --from defaults to to today",
			fromStr:  "2026-06-02",
			wantFrom: fixedFrom,
			wantTo:   t0,
		},
		{
			name:     "only --to defaults from to zero",
			toStr:    "2026-06-05",
			wantFrom: time.Time{},
			wantTo:   fixedTo,
		},
		{
			name:     "both --from and --to",
			fromStr:  "2026-06-02",
			toStr:    "2026-06-05",
			wantFrom: fixedFrom,
			wantTo:   fixedTo,
		},
		{
			name:     "--week returns monday to sunday",
			week:     true,
			wantFrom: mon,
			wantTo:   mon.AddDate(0, 0, 6),
		},
		{
			name:     "--date returns single day range",
			dateStr:  "2026-06-02",
			wantFrom: fixedFrom,
			wantTo:   fixedFrom,
		},
		{
			name:    "invalid --from returns error",
			fromStr: "not-a-date",
			wantErr: true,
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			gotFrom, gotTo, err := parseDateRange(tc.week, tc.fromStr, tc.toStr, tc.dateStr)
			if tc.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if !gotFrom.Equal(tc.wantFrom) {
				t.Errorf("from: got %v, want %v", gotFrom, tc.wantFrom)
			}
			if !gotTo.Equal(tc.wantTo) {
				t.Errorf("to: got %v, want %v", gotTo, tc.wantTo)
			}
		})
	}
}

func TestThisOrNextWeekday_SameDay(t *testing.T) {
	mon := time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC)
	if got := thisOrNextWeekday(mon, time.Monday); !got.Equal(mon) {
		t.Errorf("expected same day, got %v", got)
	}
}

func TestStrictlyNextWeekday_SameDay(t *testing.T) {
	mon := time.Date(2026, 5, 25, 0, 0, 0, 0, time.UTC)
	want := mon.AddDate(0, 0, 7)
	if got := strictlyNextWeekday(mon, time.Monday); !got.Equal(want) {
		t.Errorf("expected %v, got %v", want, got)
	}
}
