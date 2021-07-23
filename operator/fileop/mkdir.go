package fileop

import (
	"errors"
	"fmt"
	"io/fs"
	"os"

	"github.com/spf13/cobra"

	"github.com/jeffrom/polyester/operator"
	"github.com/jeffrom/polyester/operator/opfs"
	"github.com/jeffrom/polyester/state"
)

type MkdirOpts struct {
	Dests []string `json:"dests"`
	Mode  uint32   `json:"mode,omitempty"`
}

type Mkdir struct {
	Args interface{}
}

func (op Mkdir) String() string {
	opts := op.Args.(*MkdirOpts)
	return fmt.Sprintf("(Mode: %s, Dests: %v)", fs.FileMode(opts.Mode), opts.Dests)
}

func (op Mkdir) Info() operator.Info {
	opts := op.Args.(*MkdirOpts)

	cmd := &cobra.Command{
		Use:   "mkdir dir...",
		Args:  cobra.MinimumNArgs(1),
		Short: "creates or updates the permissions of dir",
		Long: `Create a directory if it doesn't already exist.

Like GNU mkdir, this will fail if a destination directory exists and is not
already a directory. Unlike GNU mkdir, this will update the mode even if the
directory already exists. Also unline GNU mkdir, parent directories are always
created (like mkdir -p).`,
	}
	flags := cmd.Flags()
	flags.Uint32VarP(&opts.Mode, "mode", "m", 0755, "the mode to set the directory to")

	return &operator.InfoData{
		OpName: "mkdir",
		Command: &operator.Command{
			Command:   cmd,
			ApplyArgs: mkdirArgs,
			Target:    opts,
		},
	}
}

func (op Mkdir) GetState(octx operator.Context) (state.State, error) {
	opts := op.Args.(*MkdirOpts)
	st := state.State{}
	for _, dest := range opts.Dests {
		info, err := octx.FS.Stat(dest)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return st, err
		}
		// fmt.Printf("info %s: %+v\n", dest, info.IsDir())
		ent := state.Entry{
			Name: dest,
			File: &opfs.StateFileEntry{
				Info: info,
			},
		}
		st = st.Append(ent.WithoutTimestamps())
	}
	return st, nil
}

func (op Mkdir) Run(octx operator.Context) error {
	opts := op.Args.(*MkdirOpts)
	mode := fs.FileMode(opts.Mode)
	for _, dest := range opts.Dests {
		info, err := octx.FS.Stat(dest)
		if err == nil {
			if info.Mode() != mode {
				if err := os.Chmod(octx.FS.Join(dest), mode); err != nil {
					return err
				}
				// could continue here, but want MkdirAll to fail if there's
				// already an existing non-directory at dest.
			}
		}

		if err := os.MkdirAll(octx.FS.Join(dest), mode); err != nil {
			return err
		}
	}
	return nil
}

func mkdirArgs(cmd *cobra.Command, args []string, target interface{}) error {
	t := target.(*MkdirOpts)
	t.Dests = args
	return nil
}
