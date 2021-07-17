package operator

import (
	"github.com/spf13/cobra"

	"github.com/jeffrom/polyester/state"
)

type Noop struct{}

var noopState = state.State{Entries: []state.Entry{
	{
		Name: "noop",
		KV:   map[string]interface{}{"noop": "ok"},
	},
}}

func (op Noop) Info() Info {
	cmd := &cobra.Command{
		Use:   "noop",
		Short: "does nothing",
		Long: `does nothing.

A caveat to this is that noop will still track an initial state change, so it
is triggered the first time it runs, which will dirty subsequent operators'
states.`,
		Args: cobra.NoArgs,
	}
	return &InfoData{
		OpName:  "noop",
		Command: &Command{Command: cmd},
	}
}

func (op Noop) GetState(octx Context) (state.State, error) {
	return noopState, nil
}

func (op Noop) Run(octx Context) error { return nil }
