package planop

import (
	"github.com/spf13/cobra"

	"github.com/jeffrom/polyester/operator"
	"github.com/jeffrom/polyester/state"
)

type DependencyOpts struct {
	Plans []string `json:"plans"`
}

type Dependency struct {
	Args interface{}
}

func (op Dependency) Info() operator.Info {
	opts := op.Args.(*DependencyOpts)

	cmd := &cobra.Command{
		Use:   "dependency plan...",
		Args:  cobra.MinimumNArgs(1),
		Short: "declares a dependency on another plan",
	}
	// flags := cmd.Flags()
	// flags.Uint32VarP(&opts.Mode, "mode", "m", 0644, "the mode to set the file to")

	return &operator.InfoData{
		OpName: "dependency",
		Command: &operator.Command{
			Command:   cmd,
			ApplyArgs: dependencyArgs,
			Target:    opts,
		},
	}
}

func (op Dependency) GetState(octx operator.Context) (state.State, error) {
	st := state.New()
	return st, nil
}

func (op Dependency) Run(octx operator.Context) error { return nil }

func dependencyArgs(cmd *cobra.Command, args []string, target interface{}) error {
	t := target.(*DependencyOpts)
	t.Plans = args
	return nil
}
