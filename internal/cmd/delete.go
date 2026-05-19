package cmd

import (
	"fmt"

	"github.com/roramirez/pitlist/internal/model"
	"github.com/spf13/cobra"
)

func newDeleteCmd() *cobra.Command {
	var force bool

	cmd := &cobra.Command{
		Use:   "delete <id>",
		Short: "Delete a task",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			task, date, err := store.GetTaskByID(id)
			if err != nil {
				return err
			}

			if !force {
				fmt.Printf("Delete %q? (y/N): ", task.Title)
				var answer string
				fmt.Scanln(&answer)
				if answer != "y" && answer != "Y" {
					fmt.Println("Aborted.")
					return nil
				}
			}

			plan, err := store.GetDayPlan(date)
			if err != nil {
				return err
			}

			remaining := make([]model.Task, 0, len(plan.Tasks))
			for _, t := range plan.Tasks {
				if t.ID != id {
					remaining = append(remaining, t)
				}
			}
			plan.Tasks = remaining

			if err := store.SaveDayPlan(plan); err != nil {
				return err
			}
			fmt.Printf("Deleted: %s\n", task.Title)
			return nil
		},
	}

	cmd.Flags().BoolVarP(&force, "force", "f", false, "skip confirmation")
	return cmd
}
