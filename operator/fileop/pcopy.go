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
	// TODO get plandir state as well
	return st, err
}

func (op Pcopy) Run(octx operator.Context) error {
	opts := op.Args.(*PcopyOpts)
	allFiles, err := gatherFilesGlobDirOnly(octx.PlanDir, opts.Sources, opts.ExcludeGlobs)
	if err != nil {
		return err
	}
	joinedFiles := make([]string, len(allFiles))
	for i, file := range allFiles {
		joinedFiles[i] = octx.PlanDir.Join(file)
	}
	return copyOneOrManyFiles(octx.PlanDir, octx.FS.Join(opts.Dest), joinedFiles)
}

func pcopyArgs(cmd *cobra.Command, args []string, target interface{}) error {
	t := target.(*PcopyOpts)
	end := len(args) - 1
	t.Sources = args[:end]
	t.Dest = args[end]
	return nil
}
