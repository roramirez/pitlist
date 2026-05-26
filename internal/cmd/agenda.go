package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/roramirez/pitlist/internal/model"
	"github.com/spf13/cobra"
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

	cmd.Flags().IntVarP(&days, "days", "n", 7, "number of days to show (default 7)")
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

func printAgenda(start, end time.Time, labels []string, showDone bool) error {
	anyFound := false
	for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
		plan, err := store.GetDayPlan(d)
		if err != nil {
			continue
		}

		var matched []model.Task
		for _, t := range plan.Tasks {
			if t.Status == model.StatusCancelled {
				continue
			}
			if t.Status == model.StatusDone && !showDone {
				continue
			}
			if !matchLabels(t.Labels, labels) {
				continue
			}
			matched = append(matched, t)
		}

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
		label += "  (today)"
	case d.Equal(t.AddDate(0, 0, 1)):
		label += "  (tomorrow)"
	case d.Before(t):
		label += "  (overdue)"
	}
	fmt.Println(strings.Repeat("─", 40))
	fmt.Printf("  %s\n", label)
	fmt.Println(strings.Repeat("─", 40))
}

func matchLabels(taskLabels, filterLabels []string) bool {
	if len(filterLabels) == 0 {
		return true
	}
	for _, want := range filterLabels {
		found := false
		for _, l := range taskLabels {
			if l == want {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}
	return true
}
