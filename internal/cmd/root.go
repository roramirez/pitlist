package cmd

import (
	"fmt"

	"github.com/roramirez/pitlist/internal/config"
	"github.com/roramirez/pitlist/internal/demo"
	"github.com/roramirez/pitlist/internal/storage"
	"github.com/roramirez/pitlist/internal/tui"
	"github.com/spf13/cobra"
)

var (
	cfg      *config.Config
	store    *storage.YAMLStore
	demoMode bool
)

func NewRootCmd() *cobra.Command {
	root := &cobra.Command{
		Use:   "pitlist",
		Short: "A CLI/TUI todo list and activity logger",
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if demoMode {
				return nil
			}
			var err error
			cfg, err = config.Load()
			if err != nil {
				return fmt.Errorf("load config: %w", err)
			}
			store, err = storage.NewYAMLStore(cfg.DataDir)
			if err != nil {
				return fmt.Errorf("open data dir: %w", err)
			}
			return nil
		},
		RunE: func(cmd *cobra.Command, args []string) error {
			if demoMode {
				dir, cleanup, err := demo.Seed()
				if err != nil {
					return fmt.Errorf("seed demo: %w", err)
				}
				defer cleanup()
				demoStore, err := storage.NewYAMLStore(dir)
				if err != nil {
					return fmt.Errorf("open demo store: %w", err)
				}
				return tui.Run(demoStore)
			}
			return tui.Run(store, cfg.Contexts...)
		},
	}

	root.Flags().BoolVar(&demoMode, "demo", false, "run with pre-seeded demo data")

	root.AddCommand(
		newAddCmd(),
		newDoneCmd(),
		newListCmd(),
		newAgendaCmd(),
		newDeleteCmd(),
		newCarryCmd(),
		newShowCmd(),
		newEditCmd(),
		newLogCmd(),
		newSyncCmd(),
		newStatsCmd(),
		newDemoSeedCmd(),
	)

	return root
}
