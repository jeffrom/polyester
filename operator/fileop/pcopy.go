package fileop

import (
	"github.com/spf13/cobra"

	"github.com/jeffrom/polyester/operator"
)

type PcopyOpts struct {
	Sources      []string `json:"sources"`
	Dest         string   `json:"dest"`
	ExcludeGlobs []string `json:"exclude,omitempty"`
}

type Pcopy struct {
	Args interface{}
}

func (op Pcopy) Info() operator.Info {
	opts := op.Args.(*PcopyOpts)

	cmd := &cobra.Command{
		Use:   "pcopy source... dest",
		Args:  cobra.MinimumNArgs(2),
		Short: "copies sources to dest",
		Long: `Manifest file copy.

Copy files, resolving paths from the plan directory.
`,
	}
	flags := cmd.Flags()
	flags.StringArrayVar(&opts.ExcludeGlobs, "exclude", nil, "`glob`s to exclude from destination")

	return &operator.InfoData{
		OpName: "pcopy",
		Command: &operator.Command{
			Command:   cmd,
			ApplyArgs: pcopyArgs,
			Target:    opts,
		},
	}
}

func (op Pcopy) GetState(octx operator.Context) (operator.State, error) {
	opts := op.Args.(*PcopyOpts)
	st, err := getStateFileGlobs(octx.FS, operator.State{}, opts.Dest, opts.Sources, opts.ExcludeGlobs)
	return st, err
}

func (op Pcopy) Run(octx operator.Context) error {
	// opts := op.Args.(*PcopyOpts)
	return nil
}

func pcopyArgs(cmd *cobra.Command, args []string, target interface{}) error {
	t := target.(*PcopyOpts)
	end := len(args) - 1
	t.Sources = args[:end]
	t.Dest = args[end]
	return nil
}