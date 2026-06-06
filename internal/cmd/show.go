package cmd

import (
	"fmt"
	"strings"

	"github.com/roramirez/pitlist/internal/model"
	"github.com/spf13/cobra"
)

func newShowCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "show <id>",
		Short: "Show task details",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			task, date, err := store.GetTaskByID(args[0])
			if err != nil {
				return err
			}

			fmt.Printf("ID:       %s\n", task.ID)
			fmt.Printf("Title:    %s\n", task.Title)
			fmt.Printf("Status:   %s\n", task.Status)
			fmt.Printf("Priority: %s\n", task.Priority)
			fmt.Printf("Date:     %s\n", date.Format(model.DateFormat))
			if len(task.Labels) > 0 {
				fmt.Printf("Labels:   %s\n", strings.Join(task.Labels, ", "))
			}
			if task.DueDate != "" {
				fmt.Printf("Due:      %s\n", task.DueDate)
			}
			if task.Notes != "" {
				fmt.Printf("Notes:\n%s\n", task.Notes)
			}
			if task.CarryFrom != "" {
				fmt.Printf("Carried from: %s\n", task.CarryFrom)
			}
			if task.CarryTo != "" {
				fmt.Printf("Carried to:   %s\n", task.CarryTo)
			}
			if len(task.Actions) > 0 {
				fmt.Println("Actions:")
				for _, a := range task.Actions {
					check := "[ ]"
					if a.Done {
						check = "[x]"
					}
					fmt.Printf("  %s %s\n", check, a.Title)
				}
			}
			return nil
		},
	}
}
