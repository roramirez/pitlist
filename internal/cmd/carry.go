package cmd

import (
	"fmt"
	"time"

	"github.com/roramirez/pitlist/internal/model"
	"github.com/roramirez/pitlist/internal/storage"
	"github.com/spf13/cobra"
)

func newCarryCmd() *cobra.Command {
	var toStr string

	cmd := &cobra.Command{
		Use:   "carry <id>",
		Short: "Move a task to another day and log it as carried",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			id := args[0]
			task, srcDate, err := store.GetTaskByID(id)
			if err != nil {
				return err
			}

			var destDate time.Time
			if toStr != "" {
				destDate, err = parseDate(toStr)
				if err != nil {
					return fmt.Errorf("invalid --to date %q: use %s", toStr, dateKeywordsHint)
				}
			} else {
				destDate = srcDate.AddDate(0, 0, 1)
			}

			if err := carryTask(store, task, srcDate, destDate); err != nil {
				return err
			}

			fmt.Printf("Carried %s → %s: %s\n", srcDate.Format(model.DateFormat), destDate.Format(model.DateFormat), task.Title)
			return nil
		},
	}

	cmd.Flags().StringVar(&toStr, "to", "", "destination date:"+dateFlag)
	return cmd
}

// carryTask moves a task from srcDate to destDate (same ID) and writes an activity log entry.
func carryTask(store *storage.YAMLStore, task *model.Task, srcDate, destDate time.Time) error {
	now := time.Now().UTC()

	// Remove from source day
	srcPlan, err := store.GetDayPlan(srcDate)
	if err != nil {
		return err
	}
	remaining := make([]model.Task, 0, len(srcPlan.Tasks))
	for _, t := range srcPlan.Tasks {
		if t.ID != task.ID {
			remaining = append(remaining, t)
		}
	}
	srcPlan.Tasks = remaining
	if err := store.SaveDayPlan(srcPlan); err != nil {
		return err
	}

	// Add to destination day (same ID, updated timestamp)
	destPlan, err := store.GetDayPlan(destDate)
	if err != nil {
		return err
	}
	moved := *task
	moved.UpdatedAt = now
	destPlan.Tasks = append(destPlan.Tasks, moved)
	if err := store.SaveDayPlan(destPlan); err != nil {
		return err
	}

	// Log activity on source date
	actLog, err := store.GetActivityLog(srcDate)
	if err != nil {
		return err
	}
	entry := model.ActivityEntry{
		ID:          storage.NextActivityID(actLog),
		Timestamp:   now,
		Description: fmt.Sprintf("Carried to %s: %s", destDate.Format(model.DateFormat), task.Title),
		Tags:        []string{"carried"},
		TaskRef:     task.ID,
	}
	actLog.Entries = append(actLog.Entries, entry)
	if err := store.SaveActivityLog(actLog); err != nil {
		return err
	}
	_ = store.AddActivityRefToTask(task.ID, model.ActivityRef{
		ID:   entry.ID,
		Date: srcDate.Format(model.DateFormat),
	})
	return nil
}
