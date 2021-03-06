// Package commands contains the available polyester cli commands.
package commands

import (
	"context"
	"errors"
	"os"

	"github.com/spf13/cobra"

	"github.com/jeffrom/polyester/compiler"
	"github.com/jeffrom/polyester/operator"
	"github.com/jeffrom/polyester/stdio"
)

func ExecArgs(ctx context.Context, args []string) error {
	rootCmd := &cobra.Command{
		Use:           "polyester",
		Args:          cobra.RangeArgs(0, 1),
		SilenceErrors: true, // we are printing errors ourselves
		SilenceUsage:  true,
		RunE: func(cmd *cobra.Command, args []string) error {
			return cmd.Usage()
		},
	}

	std := stdio.FromContext(ctx)
	rootCmd.PersistentFlags().BoolVarP(&std.Verbose, "verbose", "v", false, "Print additional debug information")
	rootCmd.PersistentFlags().BoolVarP(&std.Quiet, "quiet", "q", false, "Print only errors and warnings")

	if err := addOps(ctx, rootCmd, operatorCommandForPlan); err != nil {
		return err
	}

	execCmd, err := newExecCmd(ctx)
	if err != nil {
		return err
	}
	rootCmd.AddCommand(execCmd)

	rootCmd.AddCommand(newCheckCmd())
	rootCmd.AddCommand(newApplyCmd())

	rootCmd.SetArgs(args)
	return rootCmd.ExecuteContext(ctx)
}

func addOps(ctx context.Context, parent *cobra.Command, fn operatorCommandFunc) error {
	std := stdio.FromContext(ctx)
	for _, op := range compiler.Operators() {
		std.Debug("adding command name:", op.Info().Name())
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

	cobraCmd := cmd.Command
	cobraCmd.Hidden = true
	cobraCmd.RunE = func(cmd *cobra.Command, args []string) error {
		planFile := os.Getenv("_POLY_PLAN")
		if planFile == "" {
			return errors.New("expected $_POLY_PLAN to be set")
		}
		return compiler.AppendPlan(cmd.Context(), planFile, info, cmd, args)
	}
	return cobraCmd
}
