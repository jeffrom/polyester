package fileop

import (
	"github.com/spf13/cobra"

	"github.com/jeffrom/polyester/operator"
)

type AtomicCopyOpts struct {
	Sources      []string
	Dest         string
	ExcludeGlobs []string
}

type AtomicCopy struct {
	Args interface{}
}

func (op AtomicCopy) Info() operator.Info {
	opts := op.Args.(*AtomicCopyOpts)

	cmd := &cobra.Command{
		Use:   "atomic-copy source... dest",
		Args:  cobra.MinimumNArgs(2),
		Short: "Copies files atomically",
		Long: `Atomic file copy.

Copies file(s) into an intermediate temporary location before executing a move,
which is atomic on linux systems.`,
	}

	flags := cmd.Flags()
	flags.StringArrayVar(&opts.ExcludeGlobs, "exclude", nil, "`glob`s to exclude from destination")

	return &operator.InfoData{
		OpName: "atomic-copy",
		Command: &operator.Command{
			Command:   cmd,
			ApplyArgs: atomicCopyArgs,
			Target:    opts,
		},
	}
}

func (op AtomicCopy) GetState(octx operator.Context) (operator.State, error) {
	st := operator.State{}
	return st, nil
}

func (op AtomicCopy) Run(octx operator.Context) error {
	return nil
}

func atomicCopyArgs(cmd *cobra.Command, args []string, target interface{}) error {
	t := target.(*AtomicCopyOpts)
	end := len(args) - 1
	t.Sources = args[end-1:]
	t.Dest = args[end]
	return nil
}
