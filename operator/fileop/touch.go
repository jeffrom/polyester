package fileop

import (
	"github.com/spf13/cobra"

	"github.com/jeffrom/polyester/operator"
)

type TouchOpts struct {
	Mode uint32
	Path string
}

type Touch struct{}

func (op Touch) Name() string { return "touch" }
func (op Touch) Info() operator.Info {
	opts := &TouchOpts{}
	cmd := &cobra.Command{
		Use:   "touch FILE",
		Args:  cobra.ExactArgs(1),
		Short: "creates or updates the mtime of FILE",
		Long: `Create a file if it doesn't already exist, or update the file's
mtime if it already exists.`,
	}
	flags := cmd.Flags()
	flags.Uint32VarP(&opts.Mode, "mode", "m", 0644, "the mode to set the file to")

	return &operator.InfoData{
		Command: &operator.Command{
			Command: cmd,
			Args:    touchArgs,
			Target:  opts,
		},
	}
}

func (op Touch) GetState(octx operator.Context) (operator.State, error) {
	return operator.State{}, nil
}

func (op Touch) Run(octx operator.Context) error {

	return nil
}

func touchArgs(cmd *cobra.Command, args []string, target interface{}) error {
	t := target.(*TouchOpts)
	t.Path = args[0]
	return nil
}
