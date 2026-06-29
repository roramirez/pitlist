package cmd

import (
	"fmt"

	"github.com/roramirez/pitlist/internal/model"
	"github.com/spf13/cobra"
)

func newCloneCmd() *cobra.Command {
	var toStr string

	cmd := &cobra.Command{
		Use:   "clone <id>",
		Short: "Clone a task (with its actions, reset) to another day",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			task, srcDate, err := store.GetTaskByID(args[0])
			if err != nil {
				return err
			}

			destDate := srcDate.AddDate(0, 0, 1)
			if toStr != "" {
				destDate, err = parseDate(toStr)
				if err != nil {
					return fmt.Errorf("invalid --to date %q: use %s", toStr, dateKeywordsHint)
				}
			}

			clone, err := store.CloneTaskToDate(task, destDate)
			if err != nil {
				return err
			}

			fmt.Printf("Cloned %s → %s (%s): %s\n", task.ID, clone.ID, destDate.Format(model.DateFormat), task.Title)
			return nil
		},
	}

	cmd.Flags().StringVar(&toStr, "to", "", "destination date:"+dateFlag)
	return cmd
}
