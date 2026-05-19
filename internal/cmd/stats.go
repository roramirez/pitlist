package cmd

import (
	"fmt"
	"time"

	"github.com/roramirez/pitlist/internal/model"
	"github.com/roramirez/pitlist/internal/storage"
	"github.com/spf13/cobra"
)

func newStatsCmd() *cobra.Command {
	var week, month bool

	cmd := &cobra.Command{
		Use:   "stats",
		Short: "Show productivity stats",
		RunE: func(cmd *cobra.Command, args []string) error {
			var from, to time.Time
			now := today()

			switch {
			case month:
				from = time.Date(now.Year(), now.Month(), 1, 0, 0, 0, 0, time.UTC)
				to = now
			case week:
				from = weekStart(now)
				to = now
			default:
				from = now
				to = now
			}

			tasks, err := store.ListTasks(storage.TaskFilter{From: &from, To: &to})
			if err != nil {
				return err
			}

			var total, done, carried int
			for _, t := range tasks {
				total++
				if t.Status == model.StatusDone {
					done++
				}
				if t.CarryFrom != "" {
					carried++
				}
			}

			activities, err := store.ListActivity(storage.ActivityFilter{From: &from, To: &to})
			if err != nil {
				return err
			}

			var totalMin int
			for _, a := range activities {
				totalMin += a.DurationMin
			}

			period := from.Format("2006-01-02")
			if from != to {
				period = fmt.Sprintf("%s → %s", from.Format("2006-01-02"), to.Format("2006-01-02"))
			}

			fmt.Printf("Period:       %s\n", period)
			fmt.Printf("Tasks total:  %d\n", total)
			fmt.Printf("Tasks done:   %d", done)
			if total > 0 {
				fmt.Printf(" (%.0f%%)", float64(done)/float64(total)*100)
			}
			fmt.Println()
			fmt.Printf("Carried in:   %d\n", carried)
			fmt.Printf("Activities:   %d\n", len(activities))
			if totalMin > 0 {
				fmt.Printf("Time logged:  %dh %dm\n", totalMin/60, totalMin%60)
			}
			return nil
		},
	}

	cmd.Flags().BoolVarP(&week, "week", "w", false, "stats for this week")
	cmd.Flags().BoolVarP(&month, "month", "m", false, "stats for this month")
	return cmd
}
