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

				if _, err = pl.Apply(ctx, opts); err != nil {
					return err
				}
			}
			return nil
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&opts.DirRoot, "dir-root", "/", "use as root directory")
	flags.StringVar(&opts.StateDir, "state-dir", "/var/lib/polyester/state", "directory to track state")
	flags.StringVarP(&opts.CompiledPlan, "plan-file", "f", "", "apply a pre-compiled plan")

	return cmd
}
