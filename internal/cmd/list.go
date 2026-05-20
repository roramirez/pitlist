package cmd

import (
	"fmt"
	"strings"

	"github.com/roramirez/pitlist/internal/model"
	"github.com/roramirez/pitlist/internal/storage"
	"github.com/spf13/cobra"
)

func newListCmd() *cobra.Command {
	var labels []string
	var statuses []string
	var fromStr, toStr string
	var week bool
	var date string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List tasks",
		RunE: func(cmd *cobra.Command, args []string) error {
			filter := storage.TaskFilter{}

			if len(labels) > 0 {
				filter.Labels = labels
			}

			for _, s := range statuses {
				filter.Statuses = append(filter.Statuses, model.TaskStatus(s))
			}

			from, to, err := parseDateRange(week, fromStr, toStr, date)
			if err != nil {
				return err
			}
			filter.From = &from
			filter.To = &to

			// Default: show todo + in_progress only
			if len(filter.Statuses) == 0 {
				filter.Statuses = []model.TaskStatus{model.StatusTodo, model.StatusInProgress}
			}

			tasks, err := store.ListTasks(filter)
			if err != nil {
				return err
			}

			if len(tasks) == 0 {
				fmt.Println("No tasks found.")
				return nil
			}

			for _, t := range tasks {
				printTask(t)
			}
			return nil
		},
	}

	cmd.Flags().StringArrayVarP(&labels, "label", "l", nil, "filter by label")
	cmd.Flags().StringArrayVarP(&statuses, "status", "s", nil, "filter by status (todo|in_progress|done|cancelled)")
	cmd.Flags().StringVar(&fromStr, "from", "", "from date (YYYY-MM-DD)")
	cmd.Flags().StringVar(&toStr, "to", "", "to date (YYYY-MM-DD)")
	cmd.Flags().BoolVarP(&week, "week", "w", false, "this week")
	cmd.Flags().StringVar(&date, "date", "", "specific date (YYYY-MM-DD)")
	return cmd
}

func printTask(t *model.Task) {
	var check string
	switch t.Status {
	case model.StatusDone:
		check = "[x]"
	case model.StatusInProgress:
		check = "[~]"
	case model.StatusCancelled:
		check = "[-]"
	default:
		check = "[ ]"
	}

	labels := ""
	if len(t.Labels) > 0 {
		labels = " [" + strings.Join(t.Labels, ", ") + "]"
	}

	priority := ""
	if t.Priority == model.PriorityHigh {
		priority = " !"
	}

	carry := ""
	if t.CarryFrom != "" {
		carry = " ↑carried"
	}

	fmt.Printf("  %s %s  %s%s%s%s\n", check, t.ID, t.Title, labels, priority, carry)
}
