package fileop

import (
	"github.com/spf13/cobra"

	"github.com/jeffrom/polyester/operator"
)

type TouchOpts struct {
	Mode uint32 `json:"mode,omitempty"`
	Path string `json:"path"`
}

type Touch struct{}

func (op Touch) Info() operator.Info {
	opts := &TouchOpts{}
	cmd := &cobra.Command{
		Use:   "touch FILE",
		Args:  cobra.ExactArgs(1),
		Short: "creates or updates the mtime of FILE",
		Long: `Create a file if it doesn't already exist, otherwise update the file's
mtime`,
	}
	flags := cmd.Flags()
	flags.Uint32VarP(&opts.Mode, "mode", "m", 0644, "the mode to set the file to")

	return &operator.InfoData{
		OpName: "touch",
		Command: &operator.Command{
			Command:   cmd,
			ApplyArgs: touchArgs,
			Target:    opts,
		},
	}
}

func (op Touch) GetState(octx operator.Context) (operator.State, error) {
	st := operator.State{}
	opts := op.Info().Data().Command.Target.(*TouchOpts)
	f, err := octx.FS.Open(opts.Path)
	if err != nil {
		return st, err
	}
	info, err := octx.FS.Stat(opts.Path)
	if err != nil {
		return st, err
	}

	st = st.Append(operator.StateEntry{
		Name: opts.Path,
		File: &operator.StateFileEntry{
			File: f,
			Abs:  octx.FS.Abs(opts.Path),
			Info: info,
		},
	})
	return st, nil
}

func (op Touch) Run(octx operator.Context) error {
	return nil
}

func touchArgs(cmd *cobra.Command, args []string, target interface{}) error {
	t := target.(*TouchOpts)
	t.Path = args[0]
	return nil
}
