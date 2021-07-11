package main

import (
	"context"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jeffrom/polyester/planner"
)

func main() {
	if err := run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func run(rawArgs []string) error {
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
			_, err = pl.Reconcile(ctx)
			return err
		},
	}

	rootCmd.SetArgs(rawArgs[1:])
	return rootCmd.ExecuteContext(context.Background())
}
