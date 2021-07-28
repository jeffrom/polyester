package fileop

import (
	"github.com/spf13/cobra"

	"github.com/jeffrom/polyester/operator"
	"github.com/jeffrom/polyester/state"
)

type CopyOpts struct {
	Sources      []string `json:"sources"`
	Dest         string   `json:"dest"`
	ExcludeGlobs []string `json:"exclude,omitempty"`
}

type Copy struct {
	Args interface{}
}

func (op Copy) Info() operator.Info {
	opts := op.Args.(*CopyOpts)

	cmd := &cobra.Command{
		Use:     "copy source... dest",
		Aliases: []string{"cp"},
		Args:    cobra.MinimumNArgs(2),
		Short:   "copies sources to dest",
		Long: `Simple file copy.

See atomic-copy for copy semantics.

To copy files out of the plan directory, use pcopy.
`,
	}
	flags := cmd.Flags()
	flags.StringArrayVar(&opts.ExcludeGlobs, "exclude", nil, "`glob`s to exclude from destination")

	return &operator.InfoData{
		OpName: "copy",
		Command: &operator.Command{
			Command:   cmd,
			ApplyArgs: copyArgs,
			Target:    opts,
		},
	}
}

func (op Copy) GetState(octx operator.Context) (state.State, error) {
	opts := op.Args.(*CopyOpts)
	st, err := getStateFileGlobs(octx.FS, state.State{}, opts.Dest, opts.Sources, opts.ExcludeGlobs)
	return st, err
}

func (op Copy) Run(octx operator.Context) error {
	opts := op.Args.(*CopyOpts)
	allFiles, err := gatherFilesGlobDirOnly(octx.FS, opts.Sources, opts.ExcludeGlobs)
	if err != nil {
		return err
	}

	if err := checkPats(allFiles, opts.Sources); err != nil {
		return err
	}

	joinedFiles := make([]string, len(allFiles))
	for i, file := range allFiles {
		joinedFiles[i] = octx.FS.Join(file)
	}
	return copyOneOrManyFiles(octx.FS, octx.FS.Join(opts.Dest), joinedFiles)
}

func copyArgs(cmd *cobra.Command, args []string, target interface{}) error {
	t := target.(*CopyOpts)
	end := len(args) - 1
	t.Sources = args[:end]
	t.Dest = args[end]
	return nil
}
