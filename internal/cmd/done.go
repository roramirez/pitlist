package cmd

import (
	"fmt"
	"time"

	"github.com/roramirez/pitlist/internal/model"
	"github.com/roramirez/pitlist/internal/storage"
	"github.com/spf13/cobra"
)

func completePendingTaskIDs(_ *cobra.Command, _ []string, _ string) ([]string, cobra.ShellCompDirective) {
	cfg, err := loadConfig()
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	s, err := storage.NewYAMLStore(cfg.DataDir)
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	tasks, err := s.ListTasks(storage.TaskFilter{
		Statuses: []model.TaskStatus{model.StatusTodo, model.StatusInProgress},
	})
	if err != nil {
		return nil, cobra.ShellCompDirectiveError
	}
	ids := make([]string, 0, len(tasks))
	for _, t := range tasks {
		ids = append(ids, t.ID+"\t"+t.Title)
	}
	return ids, cobra.ShellCompDirectiveNoFileComp
}

func newDoneCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:               "done <id>",
		Short:             "Mark a task as done",
		Args:              cobra.ExactArgs(1),
		ValidArgsFunction: completePendingTaskIDs,
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
	return cmd
}
