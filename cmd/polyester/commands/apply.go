package commands

import (
	"os"

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
			var results []*planner.Result
			for _, dir := range dirs {
				pl, err := planner.New(dir)
				if err != nil {
					return err
				}

				if err := pl.Check(ctx); err != nil {
					return err
				}

				res, err := pl.Apply(ctx, opts)
				if err != nil {
					return err
				}
				if res != nil {
					results = append(results, res)
				}
			}
			for _, res := range results {
				if err := res.TextSummary(os.Stdout); err != nil {
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
