package cmd

import (
	"fmt"
	"strings"
	"time"

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

			if week {
				mon := weekStart(today())
				sun := mon.AddDate(0, 0, 6)
				filter.From = &mon
				filter.To = &sun
			} else if fromStr != "" || toStr != "" {
				if fromStr != "" {
					t, err := time.Parse("2006-01-02", fromStr)
					if err != nil {
						return fmt.Errorf("invalid --from: %w", err)
					}
					filter.From = &t
				}
				if toStr != "" {
					t, err := time.Parse("2006-01-02", toStr)
					if err != nil {
						return fmt.Errorf("invalid --to: %w", err)
					}
					filter.To = &t
				}
			} else if date != "" {
				d, err := time.Parse("2006-01-02", date)
				if err != nil {
					return fmt.Errorf("invalid --date: %w", err)
				}
				filter.From = &d
				filter.To = &d
			} else {
				d := today()
				filter.From = &d
				filter.To = &d
			}

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
	check := "[ ]"
	if t.Status == model.StatusDone {
		check = "[x]"
	} else if t.Status == model.StatusInProgress {
		check = "[~]"
	} else if t.Status == model.StatusCancelled {
		check = "[-]"
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
