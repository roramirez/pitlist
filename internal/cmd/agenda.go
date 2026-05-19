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

	cmd := &cobra.Command{
		Use:   "agenda",
		Short: "Show pending tasks grouped by day",
		RunE: func(cmd *cobra.Command, args []string) error {
			var start, end time.Time

			switch {
			case from != "" && to != "":
				var err error
				start, err = time.Parse("2006-01-02", from)
				if err != nil {
					return fmt.Errorf("invalid --from: %w", err)
				}
				end, err = time.Parse("2006-01-02", to)
				if err != nil {
					return fmt.Errorf("invalid --to: %w", err)
				}
			case from != "":
				var err error
				start, err = time.Parse("2006-01-02", from)
				if err != nil {
					return fmt.Errorf("invalid --from: %w", err)
				}
				end = start.AddDate(0, 0, days-1)
			default:
				start = today()
				end = start.AddDate(0, 0, days-1)
			}

			anyFound := false
			for d := start; !d.After(end); d = d.AddDate(0, 0, 1) {
				plan, err := store.GetDayPlan(d)
				if err != nil {
					continue
				}

				var pending []model.Task
				for _, t := range plan.Tasks {
					if t.Status == model.StatusDone || t.Status == model.StatusCancelled {
						continue
					}
					if !matchLabels(t.Labels, labels) {
						continue
					}
					pending = append(pending, t)
				}

				if len(pending) == 0 {
					continue
				}

				anyFound = true
				printDayHeader(d)
				for i := range pending {
					printTask(&pending[i])
				}
				fmt.Println()
			}

			if !anyFound {
				fmt.Println("Nothing pending.")
			}
			return nil
		},
	}

	cmd.Flags().IntVarP(&days, "days", "n", 7, "number of days to show (default 7)")
	cmd.Flags().StringArrayVarP(&labels, "label", "l", nil, "filter by label")
	cmd.Flags().StringVar(&from, "from", "", "start date (YYYY-MM-DD, default today)")
	cmd.Flags().StringVar(&to, "to", "", "end date (YYYY-MM-DD)")
	return cmd
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
