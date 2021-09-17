package commands

import (
	"github.com/spf13/cobra"

	"github.com/jeffrom/polyester/planner"
)

func newCheckCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "check [plan...]",
		Short: "check plans for validation errors",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			dirs := []string{""}
			if len(args) > 0 {
				dirs = args
			}
			for _, dir := range dirs {
				pl, err := planner.New(dir)
				if err != nil {
					return err
				}

				if err := pl.Check(ctx); err != nil {
					return err
				}
			}
			return nil
		},
	}

	// flags := cmd.Flags()
	return cmd
}
