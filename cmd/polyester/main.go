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
		Use: "polyester",
		RunE: func(cmd *cobra.Command, args []string) error {
			pl := planner.New("")
			_, err := pl.Reconcile(cmd.Context())
			return err
		},
	}

	rootCmd.SetArgs(rawArgs[1:])
	return rootCmd.ExecuteContext(context.Background())
}
