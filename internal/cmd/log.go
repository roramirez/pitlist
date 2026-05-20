package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/roramirez/pitlist/internal/model"
	"github.com/roramirez/pitlist/internal/storage"
	"github.com/spf13/cobra"
)

func newLogCmd() *cobra.Command {
	add := newLogAddCmd()

	log := &cobra.Command{
		Use:   "log [description]",
		Short: "Log an activity (or use subcommands: add, list, link)",
		Args:  cobra.ArbitraryArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if len(args) == 0 {
				return cmd.Help()
			}
			return add.RunE(cmd, args)
		},
	}

	// Inherit flags from add so `pitlist log "desc" --tag foo` works
	log.Flags().AddFlagSet(add.Flags())
	log.AddCommand(add, newLogListCmd(), newLogLinkCmd())

	return log
}

func newLogAddCmd() *cobra.Command {
	var tags []string
	var taskRef string
	var duration int
	var date string

	cmd := &cobra.Command{
		Use:   "add <description>",
		Short: "Log an activity",
		Args:  cobra.MinimumNArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			description := strings.Join(args, " ")

			var day time.Time
			var err error
			if date != "" {
				day, err = time.Parse(model.DateFormat, date)
				if err != nil {
					return fmt.Errorf("invalid date: %w", err)
				}
			} else {
				day = today()
			}

			actLog, err := store.GetActivityLog(day)
			if err != nil {
				return err
			}

			now := time.Now().UTC()
			ts := now.Add(-time.Duration(duration) * time.Minute)
			entry := model.ActivityEntry{
				ID:          storage.NextActivityID(actLog),
				Timestamp:   ts,
				Description: description,
				Tags:        tags,
				TaskRef:     taskRef,
				DurationMin: duration,
			}
			actLog.Entries = append(actLog.Entries, entry)

			if err := store.SaveActivityLog(actLog); err != nil {
				return err
			}
			if taskRef != "" {
				_ = store.AddActivityRefToTask(taskRef, model.ActivityRef{
					ID:   entry.ID,
					Date: day.Format(model.DateFormat),
				})
			}
			fmt.Printf("Logged %s: %s\n", entry.ID, entry.Description)
			return nil
		},
	}

	cmd.Flags().StringArrayVarP(&tags, "tag", "t", nil, "tag (repeatable)")
	cmd.Flags().StringVar(&taskRef, "ref", "", "link to task ID")
	cmd.Flags().IntVarP(&duration, "duration", "d", 0, "duration in minutes")
	cmd.Flags().StringVar(&date, "date", "", "log for date (YYYY-MM-DD, default today)")
	return cmd
}

func newLogListCmd() *cobra.Command {
	var tags []string
	var fromStr, toStr string
	var week bool
	var date string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List activity log entries",
		RunE: func(cmd *cobra.Command, args []string) error {
			filter := storage.ActivityFilter{Tags: tags}

			from, to, err := parseDateRange(week, fromStr, toStr, date)
			if err != nil {
				return err
			}
			filter.From = &from
			filter.To = &to

			entries, err := store.ListActivity(filter)
			if err != nil {
				return err
			}

			if len(entries) == 0 {
				fmt.Println("No activity entries found.")
				return nil
			}

			for _, e := range entries {
				printActivityEntry(e)
			}
			return nil
		},
	}

	cmd.Flags().StringArrayVarP(&tags, "tag", "t", nil, "filter by tag")
	cmd.Flags().StringVar(&fromStr, "from", "", "from date")
	cmd.Flags().StringVar(&toStr, "to", "", "to date")
	cmd.Flags().BoolVarP(&week, "week", "w", false, "this week")
	cmd.Flags().StringVar(&date, "date", "", "specific date (default today)")
	return cmd
}

func newLogLinkCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "link <activity-id> <task-id>",
		Short: "Link an activity entry to a task",
		Args:  cobra.ExactArgs(2),
		RunE: func(cmd *cobra.Command, args []string) error {
			activityID := args[0]
			taskID := args[1]

			// Verify task exists
			if _, _, err := store.GetTaskByID(taskID); err != nil {
				return fmt.Errorf("task %q: %w", taskID, err)
			}

			// Find activity entry — parse date from ID (a-YYYYMMDD-NNN)
			if len(activityID) < 11 {
				return fmt.Errorf("invalid activity ID: %q", activityID)
			}
			dateStr := activityID[2:10] // YYYYMMDD
			date, err := time.Parse("20060102", dateStr)
			if err != nil {
				return fmt.Errorf("cannot parse date from ID %q", activityID)
			}

			actLog, err := store.GetActivityLog(date)
			if err != nil {
				return err
			}

			found := false
			for i := range actLog.Entries {
				if actLog.Entries[i].ID == activityID {
					actLog.Entries[i].TaskRef = taskID
					found = true
					break
				}
			}
			if !found {
				return fmt.Errorf("activity %q not found", activityID)
			}

			if err := store.SaveActivityLog(actLog); err != nil {
				return err
			}
			_ = store.AddActivityRefToTask(taskID, model.ActivityRef{
				ID:   activityID,
				Date: date.Format(model.DateFormat),
			})
			fmt.Printf("Linked %s → %s\n", activityID, taskID)
			return nil
		},
	}
}

func printActivityEntry(e *model.ActivityEntry) {
	dur := ""
	if e.DurationMin > 0 {
		dur = fmt.Sprintf(" %dm", e.DurationMin)
	}
	tags := ""
	if len(e.Tags) > 0 {
		tags = " [" + strings.Join(e.Tags, ", ") + "]"
	}
	ref := ""
	if e.TaskRef != "" {
		ref = " → " + e.TaskRef
	}
	fmt.Printf("  %s  %s%s%s%s\n",
		e.Timestamp.Local().Format("15:04"),
		e.Description,
		dur, tags, ref,
	)
}
