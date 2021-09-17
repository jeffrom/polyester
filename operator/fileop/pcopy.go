package fileop

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jeffrom/polyester/operator"
	"github.com/jeffrom/polyester/operator/opfs"
	"github.com/jeffrom/polyester/state"
	"github.com/jeffrom/polyester/stdio"
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
	// TODO mode

	return &operator.InfoData{
		OpName: "pcopy",
		Command: &operator.Command{
			Command:   cmd,
			ApplyArgs: pcopyArgs,
			Target:    opts,
		},
	}
}

func (op Pcopy) GetState(octx operator.Context) (state.State, error) {
	std := stdio.FromContext(octx.Context)
	std.Debug("pcopy: GetState")
	opts := op.Args.(*PcopyOpts)
	// TODO ResolvePlanFile to get source (plan files) state, get dest state as
	// normal
	st := state.State{}
	sources, err := octx.PlanDir.Resolve("files", opts.Sources)
	if err != nil {
		return st, err
	}
	std.Debug("pcopy: GetState: source files:", sources)
	st, err = appendFiles(opfs.New("/"), st, true, false, sources...)
	if err != nil {
		return st, err
	}

	st, err = getStateFileGlobs(octx.FS, st, opts.Dest, sources, opts.ExcludeGlobs)
	if err != nil {
		return st, err
	}
	st = st.Map(func(e state.Entry) state.Entry { return e.ChecksumOnly() })
	if std.Verbose {
		st.WriteTo(std.Stdout())
	}
	return st, nil
}

// func (op Pcopy) Changed(a, b state.State) (bool, error) {
// 	fn := func(e state.Entry) state.Entry { return e.WithoutTimestamps() }
// 	return a.Map(fn).Changed(b.Map(fn)), nil
// }

func (op Pcopy) Run(octx operator.Context) error {
	opts := op.Args.(*PcopyOpts)
	sources, err := octx.PlanDir.Resolve("files", opts.Sources)
	if err != nil {
		return err
	}

	// check that each pattern matches
	for _, pat := range opts.Sources {
		sources, err := octx.PlanDir.Resolve("files", []string{pat})
		if err != nil {
			return err
		}
		if len(sources) == 0 {
			return fmt.Errorf("pcopy: pattern %q did not match any file(s)", pat)
		}
	}

	joinedFiles := make([]string, len(sources))
	for i, file := range sources {
		joinedFiles[i] = octx.PlanDir.Join(file)
	}
	// fmt.Println("copyOneOrManyFiles to:", opts.Dest, octx.FS.Join(opts.Dest))
	return copyOneOrManyFiles(octx.PlanDir, octx.FS.Join(opts.Dest), joinedFiles)
}

func pcopyArgs(cmd *cobra.Command, args []string, target interface{}) error {
	t := target.(*PcopyOpts)
	end := len(args) - 1
	t.Sources = args[:end]
	t.Dest = args[end]
	return nil
}
