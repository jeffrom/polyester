package commands

import (
	"github.com/spf13/cobra"

	"github.com/jeffrom/polyester/planner"
)

func newApplyCmd() *cobra.Command {
	opts := planner.ApplyOpts{}
	cmd := &cobra.Command{
		Use:   "apply [plan...]",
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

			_, err = pl.Apply(ctx, opts)
			return err
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&opts.DirRoot, "dir-root", "/", "use as root directory")
	flags.StringVarP(&opts.Plan, "plan-file", "f", "", "apply a pre-compiled plan")

	return cmd
}
