package fileop

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/spf13/cobra"

	"github.com/jeffrom/polyester/operator"
	"github.com/jeffrom/polyester/operator/opfs"
)

type TouchOpts struct {
	Mode uint32 `json:"mode,omitempty"`
	Path string `json:"path"`
}

type Touch struct {
	Args interface{}
}

func (op Touch) Info() operator.Info {
	opts := op.Args.(*TouchOpts)

	cmd := &cobra.Command{
		Use:   "touch file",
		Args:  cobra.ExactArgs(1),
		Short: "creates or updates the mtime of file",
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
	opts := op.Args.(*TouchOpts)
	st := operator.State{}
	fmt.Printf("touch: GetState opts: %+v\n", opts)
	info, err := octx.FS.Stat(opts.Path)
	if err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return st, err
		}
	}

	st = st.Append(operator.StateEntry{
		Name: opts.Path,
		File: &opfs.StateFileEntry{
			// File: f,
			Info: info,
		},
	})
	return st, nil
}

func (op Touch) Run(octx operator.Context) error {
	opts := op.Args.(*TouchOpts)
	_, err := octx.FS.Stat(opts.Path)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	} else if err == nil {
		return nil
	}

	dest := octx.FS.Join(opts.Path)
	dir, _ := filepath.Split(dest)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	f, err := os.OpenFile(dest, os.O_CREATE, fs.FileMode(opts.Mode))
	if err != nil {
		return err
	}
	if err := f.Close(); err != nil {
		return err
	}
	return nil
}

func touchArgs(cmd *cobra.Command, args []string, target interface{}) error {
	t := target.(*TouchOpts)
	t.Path = args[0]
	return nil
}
