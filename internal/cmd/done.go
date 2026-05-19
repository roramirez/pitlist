package cmd

import (
	"fmt"
	"time"

	"github.com/roramirez/pitlist/internal/model"
	"github.com/spf13/cobra"
)

func newDoneCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "done <id>",
		Short: "Mark a task as done",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			task, date, err := store.GetTaskByID(id)
			if err != nil {
				return err
			}

			plan, err := store.GetDayPlan(date)
			if err != nil {
				return err
			}

			now := time.Now().UTC()
			for i := range plan.Tasks {
				if plan.Tasks[i].ID == id {
					plan.Tasks[i].Status = model.StatusDone
					plan.Tasks[i].DoneAt = &now
					plan.Tasks[i].UpdatedAt = now
					break
				}
			}

			if err := store.SaveDayPlan(plan); err != nil {
				return err
			}
			fmt.Printf("Done: %s\n", task.Title)
			return nil
		},
	}
}
