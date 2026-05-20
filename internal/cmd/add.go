package cmd

import (
	"fmt"
	"time"

	"github.com/roramirez/pitlist/internal/model"
	"github.com/roramirez/pitlist/internal/storage"
	"github.com/spf13/cobra"
)

func newAddCmd() *cobra.Command {
	var labels []string
	var priority string
	var due string
	var date string
	var context string

	cmd := &cobra.Command{
		Use:   "add <title>",
		Short: "Add a task",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			title := args[0]

			var day time.Time
			var err error
			if date != "" {
				day, err = time.Parse(model.DateFormat, date)
				if err != nil {
					return fmt.Errorf("invalid date %q: use YYYY-MM-DD", date)
				}
			} else {
				day = today()
			}

			p := model.Priority(priority)
			if p != model.PriorityLow && p != model.PriorityMedium && p != model.PriorityHigh {
				p = model.PriorityMedium
			}

			plan, err := store.GetDayPlan(day)
			if err != nil {
				return err
			}

			now := time.Now().UTC()
			task := model.Task{
				ID:        storage.NextTaskID(plan),
				Title:     title,
				Context:   context,
				Labels:    labels,
				Status:    model.StatusTodo,
				Priority:  p,
				CreatedAt: now,
				UpdatedAt: now,
				DueDate:   due,
			}
			plan.Tasks = append(plan.Tasks, task)

			if err := store.SaveDayPlan(plan); err != nil {
				return err
			}
			fmt.Printf("Added %s: %s\n", task.ID, task.Title)
			return nil
		},
	}

	cmd.Flags().StringArrayVarP(&labels, "label", "l", nil, "label (repeatable)")
	cmd.Flags().StringVarP(&priority, "priority", "p", "medium", "priority: low|medium|high")
	cmd.Flags().StringVarP(&context, "context", "c", "", "context: work|personal|other")
	cmd.Flags().StringVar(&due, "due", "", "due date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&date, "date", "", "plan for this date (YYYY-MM-DD, default today)")
	return cmd
}
