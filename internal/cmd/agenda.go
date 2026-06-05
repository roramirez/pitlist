package cmd

import (
	"fmt"
	"slices"
	"strings"
	"time"

	"github.com/roramirez/pitlist/internal/model"
	"github.com/spf13/cobra"
)

const defaultAgendaDays = 7

const (
	agendaLabelToday    = "  (today)"
	agendaLabelTomorrow = "  (tomorrow)"
	agendaSeparatorLen  = 40
)

func newAgendaCmd() *cobra.Command {
	var days int
	var labels []string
	var from, to string
	var showDone bool

	cmd := &cobra.Command{
		Use:   "agenda",
		Short: "Show pending tasks grouped by day",
		RunE: func(cmd *cobra.Command, args []string) error {
			start, end, err := resolveAgendaRange(from, to, days)
			if err != nil {
				return err
			}
			return printAgenda(start, end, labels, showDone)
		},
	}

	cmd.Flags().IntVarP(&days, "days", "n", defaultAgendaDays, "number of days to show")
	cmd.Flags().StringArrayVarP(&labels, "label", "l", nil, "filter by label")
	cmd.Flags().StringVar(&from, "from", "", "start date:"+dateFlag)
	cmd.Flags().StringVar(&to, "to", "", "end date:"+dateFlag)
	cmd.Flags().BoolVar(&showDone, "done", false, "include completed tasks")
	return cmd
}

func resolveAgendaRange(from, to string, days int) (start, end time.Time, err error) {
	switch {
	case from != "" && to != "":
		start, err = parseDate(from)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid --from %q: use %s", from, dateKeywordsHint)
		}
		end, err = parseDate(to)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid --to %q: use %s", to, dateKeywordsHint)
		}
	case from != "":
		start, err = parseDate(from)
		if err != nil {
			return time.Time{}, time.Time{}, fmt.Errorf("invalid --from %q: use %s", from, dateKeywordsHint)
		}
		end = start.AddDate(0, 0, days-1)
	default:
		start = today()
		end = start.AddDate(0, 0, days-1)
	}
	return start, end, nil
}

func filterAgendaTasks(tasks []model.Task, labels []string, showDone bool) []model.Task {
	var out []model.Task
	for _, t := range tasks {
		if t.Status == model.StatusCancelled {
			continue
		}
		if t.Status == model.StatusDone && !showDone {
			continue
		}
		if !matchLabels(t.Labels, labels) {
			continue
		}
		out = append(out, t)
	}
	return out
}

func printAgenda(start, end time.Time, labels []string, showDone bool) error {
	anyFound := false
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		plan, err := store.GetDayPlan(d)
		if err != nil {
			continue
		}
		matched := filterAgendaTasks(plan.Tasks, labels, showDone)
		if len(matched) == 0 {
			continue
		}
		anyFound = true
		printDayHeader(d)
		for i := range matched {
			printTask(&matched[i])
		}
		fmt.Println()
	}

	if !anyFound {
		if showDone {
			fmt.Println("Nothing found.")
		} else {
			fmt.Println("Nothing pending.")
		}
	}
	return nil
}

func printDayHeader(d time.Time) {
	t := today()
	label := d.Format("Mon 2006-01-02")
	switch {
	case d.Equal(t):
		label += agendaLabelToday
	case d.Equal(t.AddDate(0, 0, 1)):
		label += agendaLabelTomorrow
	}
	sep := strings.Repeat("─", agendaSeparatorLen)
	fmt.Println(sep)
	fmt.Printf("  %s\n", label)
	fmt.Println(sep)
}

func matchLabels(taskLabels, filterLabels []string) bool {
	for _, want := range filterLabels {
		if !slices.Contains(taskLabels, want) {
			return false
		}
	}
	return true
}
