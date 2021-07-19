package templateop

import (
	"bytes"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/jeffrom/polyester/operator"
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
	fmt.Printf("template: GetState opts: %+v\n", opts)

	b := &bytes.Buffer{}
	if err := octx.Templates.ExecuteForOp(b, opts.Path, templates.Data{}); err != nil {
		return st, err
	}
	fmt.Println("rendered:", b.String())
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
