package cmd

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/roramirez/pitlist/internal/storage"
	"github.com/spf13/cobra"
)

func newSyncCmd() *cobra.Command {
	var push bool

	cmd := &cobra.Command{
		Use:   "sync",
		Short: "Commit all changes in the data directory",
		RunE: func(cmd *cobra.Command, args []string) error {
			dataDir := cfg.DataDir

			add := exec.Command("git", "-C", dataDir, "add", ".")
			add.Stdout = os.Stdout
			add.Stderr = os.Stderr
			if err := add.Run(); err != nil {
				return fmt.Errorf("git add: %w", err)
			}

			commit := exec.Command("git", "-C", dataDir, "commit", "-m", "pitlist: manual sync")
			commit.Env = storage.GitEnv()
			commit.Stdout = os.Stdout
			commit.Stderr = os.Stderr
			if err := commit.Run(); err != nil {
				// exit code 1 means "nothing to commit" — not an error
				fmt.Println("Nothing new to commit.")
			}

			if push {
				p := exec.Command("git", "-C", dataDir, "push")
				p.Stdout = os.Stdout
				p.Stderr = os.Stderr
				if err := p.Run(); err != nil {
					return fmt.Errorf("git push: %w", err)
				}
			}
			return nil
		},
	}

	cmd.Flags().BoolVar(&push, "push", false, "also push to remote")
	return cmd
}
