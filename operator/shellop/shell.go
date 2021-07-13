package shellop

import (
	"os"
	"os/exec"

	"github.com/spf13/cobra"

	"github.com/jeffrom/polyester/operator"
)

type ShellOpts struct {
	Dir    string `json:"dir"`
	Script string `json:"script"`
}

type Shell struct {
	Args interface{}
}

func (op Shell) Info() operator.Info {
	opts := op.Args.(*ShellOpts)

	cmd := &cobra.Command{
		Use:   "sh script",
		Args:  cobra.ExactArgs(1),
		Short: "executes a shell script",
	}
	flags := cmd.Flags()
	flags.StringVar(&opts.Dir, "dir", "", "the directory to run the script in")

	return &operator.InfoData{
		OpName: "sh",
		Command: &operator.Command{
			Command:   cmd,
			ApplyArgs: shellArgs,
			Target:    opts,
		},
	}
}

func (op Shell) GetState(octx operator.Context) (operator.State, error) {
	st := operator.State{}
	return st, nil
}

func (op Shell) Run(octx operator.Context) error {
	opts := op.Args.(*ShellOpts)
	args := []string{
		"-c", opts.Script,
	}
	cmd := exec.CommandContext(octx.Context, "sh", args...)
	cmd.Dir = opts.Dir
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func shellArgs(cmd *cobra.Command, args []string, target interface{}) error {
	t := target.(*ShellOpts)
	t.Script = args[0]
	return nil
}
