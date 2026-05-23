package cmd

import (
	"fmt"

	"github.com/roramirez/pitlist/internal/demo"
	"github.com/roramirez/pitlist/internal/storage"
	"github.com/spf13/cobra"
)

func newDemoSeedCmd() *cobra.Command {
	return &cobra.Command{
		Use:    "demo-seed <dir>",
		Short:  "Seed demo data into <dir> and exit",
		Hidden: true,
		Args:   cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			dir := args[0]
			s, err := storage.NewYAMLStore(dir)
			if err != nil {
				return fmt.Errorf("open store: %w", err)
			}
			return demo.SeedInto(s)
		},
	}
}
