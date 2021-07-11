// Package commands contains the available polyester cli commands.
package commands

import (
	"context"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jeffrom/polyester/operator"
	"github.com/jeffrom/polyester/planner"
)

func ExecArgs(ctx context.Context, args []string) error {
	rootCmd := &cobra.Command{
		Use:  "polyester",
		Args: cobra.RangeArgs(0, 1),
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
			return nil
			// _, err = pl.Reconcile(ctx)
			// return err
		},
	}

	if err := addOps(rootCmd, operatorCommandForPlan); err != nil {
		return err
	}

	execCmd, err := newExecCmd()
	if err != nil {
		return err
	}
	rootCmd.AddCommand(execCmd)

	rootCmd.AddCommand(newApplyCmd())

	rootCmd.SetArgs(args)
	return rootCmd.ExecuteContext(ctx)
}

func addOps(parent *cobra.Command, fn operatorCommandFunc) error {
	for _, op := range planner.Operators() {
		fmt.Println("adding command name:", op.Name())
		parent.AddCommand(fn(op))
	}
	return nil
}

type operatorCommandFunc func(op operator.Interface) *cobra.Command

// operatorCommandForPlan commands, when run, are only written into a plan
// file. These are the commands that are called in plan scripts.
func operatorCommandForPlan(op operator.Interface) *cobra.Command {
	info := op.Info()
	cmd := info.Data().Command
	// target := cmd.Target

	cobraCmd := &*cmd.Command
	cobraCmd.Hidden = true
	cobraCmd.RunE = func(cmd *cobra.Command, args []string) error {
		return nil
	}
	return cobraCmd
}
