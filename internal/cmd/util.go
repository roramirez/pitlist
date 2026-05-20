package cmd

import (
	"fmt"
	"time"

	"github.com/roramirez/pitlist/internal/model"
)

func today() time.Time {
	t := time.Now()
	return time.Date(t.Year(), t.Month(), t.Day(), 0, 0, 0, 0, time.UTC)
}

func weekStart(t time.Time) time.Time {
	wd := t.Weekday()
	if wd == time.Sunday {
		wd = 7
	}
	return t.AddDate(0, 0, -int(wd-time.Monday))
}

// parseDateRange resolves --week / --from / --to / --date flags into a concrete date range.
func parseDateRange(week bool, fromStr, toStr, dateStr string) (from, to time.Time, err error) {
	if week {
		mon := weekStart(today())
		return mon, mon.AddDate(0, 0, 6), nil
	}
	if fromStr != "" {
		if from, err = time.Parse(model.DateFormat, fromStr); err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid --from: %w", err)
		}
	}
	if toStr != "" {
		if to, err = time.Parse(model.DateFormat, toStr); err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid --to: %w", err)
		}
	}
	if fromStr != "" || toStr != "" {
		return from, to, nil
	}
	if dateStr != "" {
		if from, err = time.Parse(model.DateFormat, dateStr); err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid --date: %w", err)
		}
		return from, from, nil
	}
	d := today()
	return d, d, nil
}
