package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/spf13/cobra"
)

func newEditCmd() *cobra.Command {
	return &cobra.Command{
		Use:   "edit <id>",
		Short: "Open task day file in $EDITOR",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			_, date, err := store.GetTaskByID(args[0])
			if err != nil {
				return err
			}

			editor := cfg.Editor
			if editor == "" {
				editor = os.Getenv("EDITOR")
			}
			if editor == "" {
				return fmt.Errorf("no editor set — define $EDITOR or editor in config")
			}

			dayFile := fmt.Sprintf("%s/days/%s.yaml", cfg.DataDir, date.Format("2006-01-02"))
			c := exec.Command(editor, dayFile)
			c.Stdin = os.Stdin
			c.Stdout = os.Stdout
			c.Stderr = os.Stderr
			return c.Run()
		},
	}
}
