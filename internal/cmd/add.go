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
	var future bool

	cmd := &cobra.Command{
		Use:   "add <title>",
		Short: "Add a task",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			title := args[0]

			p := model.Priority(priority)
			if p != model.PriorityLow && p != model.PriorityMedium && p != model.PriorityHigh {
				p = model.PriorityMedium
			}

			now := time.Now().UTC()

			if future {
				list, err := store.GetFutureList()
				if err != nil {
					return err
				}
				task := model.Task{
					ID:        storage.NextFutureTaskID(list),
					Title:     title,
					Context:   context,
					Labels:    labels,
					Status:    model.StatusTodo,
					Priority:  p,
					CreatedAt: now,
					UpdatedAt: now,
				}
				list.Tasks = append(list.Tasks, task)
				if err := store.SaveFutureList(list); err != nil {
					return err
				}
				fmt.Printf("Added %s: %s\n", task.ID, task.Title)
				return nil
			}

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

			plan, err := store.GetDayPlan(day)
			if err != nil {
				return err
			}

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
	cmd.Flags().StringVar(&date, "date", "", "plan for this date:"+dateFlag)
	cmd.Flags().BoolVar(&future, "future", false, "add to future backlog (no date)")
	return cmd
}
