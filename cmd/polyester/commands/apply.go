package commands

import (
	"github.com/spf13/cobra"

	"github.com/jeffrom/polyester/planner"
)

func newApplyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply [PLAN...]",
		Short: "read, check, and execute plans",
		RunE: func(cmd *cobra.Command, args []string) error {
			ctx := cmd.Context()
			dir := ""
			if len(args) > 0 {
				dir = args[0]
			}

			pl, err := planner.New(dir)
			if err != nil {
				return err
			}

			if err := pl.Check(ctx); err != nil {
				return err
			}

			_, err = pl.Apply(ctx)
			return err
		},
	}

	return cmd
}
