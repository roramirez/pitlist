package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/roramirez/pitlist/internal/model"
)

const dateKeywordsHint = "YYYY-MM-DD or a keyword: today, tomorrow, yesterday, " +
	"next_week, last_week, in_a_week, next_month, last_month, in_a_month, " +
	"monday…sunday (this week's day), next_monday…next_sunday (strictly next)"

// dateFlag is the shared flag description for any date flag, formatted for cobra --help output.
const dateFlag = "\n" +
	"  YYYY-MM-DD\n" +
	"  today, tomorrow, yesterday\n" +
	"  next_week, last_week, in_a_week\n" +
	"  next_month, last_month, in_a_month\n" +
	"  monday…sunday  (upcoming, incl. today)\n" +
	"  next_monday…next_sunday  (strictly next)"

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

// thisOrNextWeekday returns t if its weekday matches wd, otherwise the next occurrence.
func thisOrNextWeekday(t time.Time, wd time.Weekday) time.Time {
	days := int(wd) - int(t.Weekday())
	if days < 0 {
		days += 7
	}
	return t.AddDate(0, 0, days)
}

// strictlyNextWeekday always returns the next occurrence of wd after t (never t itself).
func strictlyNextWeekday(t time.Time, wd time.Weekday) time.Time {
	days := int(wd) - int(t.Weekday())
	if days <= 0 {
		days += 7
	}
	return t.AddDate(0, 0, days)
}

// parseDate parses a YYYY-MM-DD string or a natural-language keyword into a time.Time.
func parseDate(s string) (time.Time, error) {
	t := today()
	switch strings.ToLower(s) {
	case "today":
		return t, nil
	case "tomorrow":
		return t.AddDate(0, 0, 1), nil
	case "yesterday":
		return t.AddDate(0, 0, -1), nil
	case "next_week":
		return weekStart(t).AddDate(0, 0, 7), nil
	case "last_week":
		return weekStart(t).AddDate(0, 0, -7), nil
	case "in_a_week":
		return t.AddDate(0, 0, 7), nil
	case "next_month":
		return time.Date(t.Year(), t.Month()+1, 1, 0, 0, 0, 0, t.Location()), nil
	case "last_month":
		return time.Date(t.Year(), t.Month()-1, 1, 0, 0, 0, 0, t.Location()), nil
	case "in_a_month":
		return t.AddDate(0, 0, 30), nil
	case "monday":
		return thisOrNextWeekday(t, time.Monday), nil
	case "tuesday":
		return thisOrNextWeekday(t, time.Tuesday), nil
	case "wednesday":
		return thisOrNextWeekday(t, time.Wednesday), nil
	case "thursday":
		return thisOrNextWeekday(t, time.Thursday), nil
	case "friday":
		return thisOrNextWeekday(t, time.Friday), nil
	case "saturday":
		return thisOrNextWeekday(t, time.Saturday), nil
	case "sunday":
		return thisOrNextWeekday(t, time.Sunday), nil
	case "next_monday":
		return strictlyNextWeekday(t, time.Monday), nil
	case "next_tuesday":
		return strictlyNextWeekday(t, time.Tuesday), nil
	case "next_wednesday":
		return strictlyNextWeekday(t, time.Wednesday), nil
	case "next_thursday":
		return strictlyNextWeekday(t, time.Thursday), nil
	case "next_friday":
		return strictlyNextWeekday(t, time.Friday), nil
	case "next_saturday":
		return strictlyNextWeekday(t, time.Saturday), nil
	case "next_sunday":
		return strictlyNextWeekday(t, time.Sunday), nil
	default:
		return time.Parse(model.DateFormat, s)
	}
}

// parseDateRange resolves --week / --from / --to / --date flags into a concrete date range.
func parseDateRange(week bool, fromStr, toStr, dateStr string) (from, to time.Time, err error) {
	if week {
		mon := weekStart(today())
		return mon, mon.AddDate(0, 0, 6), nil
	}
	if fromStr != "" {
		if from, err = parseDate(fromStr); err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid --from: %w", err)
		}
	}
	if toStr != "" {
		if to, err = parseDate(toStr); err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid --to: %w", err)
		}
	}
	if fromStr != "" || toStr != "" {
		return from, to, nil
	}
	if dateStr != "" {
		if from, err = parseDate(dateStr); err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid --date: %w", err)
		}
		return from, from, nil
	}
	d := today()
	return d, d, nil
}
