package templateop

import (
	"bytes"
	"fmt"
	"os"

	"github.com/spf13/cobra"

	"github.com/jeffrom/polyester/operator"
	"github.com/jeffrom/polyester/operator/fileop"
	"github.com/jeffrom/polyester/operator/opfs"
	"github.com/jeffrom/polyester/operator/templates"
	"github.com/jeffrom/polyester/state"
)

type TemplateOpts struct {
	Path      string   `json:"path"`
	Dests     []string `json:"dests"`
	DataPaths []string `json:"data,omitempty"`
}

type Template struct {
	Args interface{}
}

func (op Template) Info() operator.Info {
	opts := op.Args.(*TemplateOpts)

	cmd := &cobra.Command{
		Use:   "template template dest...",
		Args:  cobra.MinimumNArgs(2),
		Short: "Renders a template to file(s)",
	}
	flags := cmd.Flags()
	flags.StringArrayVarP(&opts.DataPaths, "data", "d", nil, "template data `file`(s)")
	// flags.Uint32VarP(&opts.Mode, "mode", "m", 0644, "the mode to set the file to")

	return &operator.InfoData{
		OpName: "template",
		Command: &operator.Command{
			Command:   cmd,
			ApplyArgs: templateArgs,
			Target:    opts,
		},
	}
}

func (op Template) GetState(octx operator.Context) (state.State, error) {
	opts := op.Args.(*TemplateOpts)
	st := state.State{}

	dataPaths, err := octx.PlanDir.Resolve("vars", opts.DataPaths)
	if err != nil {
		return st, err
	}
	absDataPaths := make([]string, len(dataPaths))
	for i, dataPath := range dataPaths {
		absDataPaths[i] = octx.PlanDir.Join(dataPath)
	}

	defaultVarsPath := octx.PlanDir.Join("vars", "default.yaml")
	if _, err := os.Stat(defaultVarsPath); err == nil {
		absDataPaths = append([]string{defaultVarsPath}, absDataPaths...)
	}
	userData, err := octx.Templates.MergeData(absDataPaths)
	if err != nil {
		return st, err
	}
	fmt.Printf("template: GetState opts: %+v\ndata:%+v\n", opts, userData)

	for i, dest := range opts.Dests {
		buf := &bytes.Buffer{}
		data := templates.Data{
			Data:    userData,
			Dest:    dest,
			DestIdx: i,
		}
		if err := octx.Templates.ExecuteForOp(buf, opts.Path, data); err != nil {
			return st, err
		}
		b := buf.Bytes()

		checksum, err := fileop.ChecksumReader(bytes.NewReader(b))
		if err != nil {
			return st, err
		}
		// fmt.Printf("checksum %s, rendered:\n%s\n", string(checksum), string(b))

		st = st.Append(state.Entry{
			Name: dest,
			File: &opfs.StateFileEntry{
				SHA256: checksum,
			},
		})
	}
	return st, nil
}

func (op Template) Run(octx operator.Context) error {
	// opts := op.Args.(*TemplateOpts)
	return nil
}

func templateArgs(cmd *cobra.Command, args []string, target interface{}) error {
	t := target.(*TemplateOpts)
	t.Path = args[0]
	t.Dests = args[1:]
	return nil
}
