package cmd

import (
	"fmt"
	"time"

	"github.com/roramirez/pitlist/internal/model"
	"github.com/roramirez/pitlist/internal/storage"
	"github.com/spf13/cobra"
)

func newScheduleCmd() *cobra.Command {
	var date string

	cmd := &cobra.Command{
		Use:   "schedule <future-task-id>",
		Short: "Move a future task to a specific day",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]

			var day time.Time
			var err error
			if date != "" {
				day, err = parseDate(date)
				if err != nil {
					return fmt.Errorf("invalid date %q: use %s", date, dateKeywordsHint)
				}
			} else {
				day = today()
			}

			list, err := store.GetFutureList()
			if err != nil {
				return err
			}

			var task *model.Task
			remaining := make([]model.Task, 0, len(list.Tasks))
			for i := range list.Tasks {
				if list.Tasks[i].ID == id {
					t := list.Tasks[i]
					task = &t
				} else {
					remaining = append(remaining, list.Tasks[i])
				}
			}
			if task == nil {
				return fmt.Errorf("future task %q not found", id)
			}

			list.Tasks = remaining
			if err := store.SaveFutureList(list); err != nil {
				return err
			}

			plan, err := store.GetDayPlan(day)
			if err != nil {
				return err
			}
			moved := *task
			moved.ID = storage.NextTaskID(plan)
			moved.UpdatedAt = time.Now().UTC()
			plan.Tasks = append(plan.Tasks, moved)
			if err := store.SaveDayPlan(plan); err != nil {
				return err
			}

			fmt.Printf("Scheduled %s → %s (%s)\n", id, moved.ID, day.Format(model.DateFormat))
			return nil
		},
	}

	cmd.Flags().StringVar(&date, "date", "", "target date (default: today):"+dateFlag)
	return cmd
}
