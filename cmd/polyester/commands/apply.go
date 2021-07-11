package commands

import "github.com/spf13/cobra"

func newApplyCmd() *cobra.Command {
	cmd := &cobra.Command{
		Use:   "apply [PLAN...]",
		Short: "read, check, and execute plans",
		RunE: func(cmd *cobra.Command, args []string) error {

			return nil
		},
	}

	return cmd
}
