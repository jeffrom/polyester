package planop

import (
	"github.com/spf13/cobra"

	"github.com/jeffrom/polyester/operator"
	"github.com/jeffrom/polyester/state"
)

type PlanOpts struct {
	Plans []string `json:"plans"`
}

type Plan struct {
	Args interface{}
}

func (op Plan) Info() operator.Info {
	opts := op.Args.(*PlanOpts)

	cmd := &cobra.Command{
		Use:   "plan plan...",
		Args:  cobra.MinimumNArgs(1),
		Short: "execute a plan as part of another plan",
	}
	// flags := cmd.Flags()
	// flags.Uint32VarP(&opts.Mode, "mode", "m", 0644, "the mode to set the file to")

	return &operator.InfoData{
		OpName: "plan",
		Command: &operator.Command{
			Command:   cmd,
			ApplyArgs: planArgs,
			Target:    opts,
		},
	}
}

func (op Plan) GetState(octx operator.Context) (state.State, error) {
	st := state.State{}
	return st, nil
}

func (op Plan) Run(octx operator.Context) error { return nil }

func planArgs(cmd *cobra.Command, args []string, target interface{}) error {
	t := target.(*PlanOpts)
	t.Plans = args
	return nil
}
