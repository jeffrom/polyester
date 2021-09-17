package commands

import (
	"context"

	"github.com/spf13/cobra"

	"github.com/jeffrom/polyester/operator"
)

func newExecCmd(ctx context.Context) (*cobra.Command, error) {
	cmd := &cobra.Command{
		Use:    "exec",
		Short:  "executes operators",
		Hidden: true,
	}

	if err := addOps(ctx, cmd, operatorCommandForExec); err != nil {
		return nil, err
	}
	return cmd, nil
}

func operatorCommandForExec(op operator.Interface) *cobra.Command {
	info := op.Info()
	cmd := info.Data().Command

	cobraCmd := cmd.Command
	cobraCmd.RunE = func(cmd *cobra.Command, args []string) error {
		return nil
	}
	return cobraCmd
}
