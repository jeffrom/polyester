package templateop

import (
	"bytes"
	"errors"
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

	for _, dest := range opts.Dests {
		info, err := octx.FS.Stat(dest)
		if err != nil && !errors.Is(err, os.ErrNotExist) {
			return st, err
		}

		var checksum []byte
		if info != nil && !info.IsDir() {
			var err error
			checksum, err = fileop.Checksum(octx.FS.Join(dest))
			if err != nil {
				return st, err
			}
		}
		// fmt.Printf("%s: checksum %s\n", octx.FS.Join(dest), string(checksum))

		var fi *opfs.StateFileEntry
		if len(checksum) > 0 {
			fi = &opfs.StateFileEntry{
				SHA256: checksum,
				Info: opfs.StateFileInfo{
					RawName: info.Name(),
					SHA256:  checksum,
				},
			}
		}
		st = st.Append(state.Entry{
			Name: dest,
			File: fi,
		})
	}
	// st.WriteTo(os.Stdout)
	// println()
	return st, nil
}

func (op Template) DesiredState(octx operator.Context) (state.State, error) {
	return state.New(), nil
	// opts := op.Args.(*TemplateOpts)
	// st := state.State{}

	// userData, err := readUserData(octx, opts)
	// if err != nil {
	// 	return st, err
	// }
	// // fmt.Printf("template: GetState opts: %+v\ndata:%+v\n", opts, userData)

	// for i, dest := range opts.Dests {
	// 	b, err := executeTemplate(octx, opts.Path, dest, i, userData)
	// 	if err != nil {
	// 		return st, err
	// 	}

	// 	checksum, err := fileop.ChecksumReader(bytes.NewReader(b))
	// 	if err != nil {
	// 		return st, err
	// 	}
	// 	// fmt.Printf("checksum %s, rendered:\n%s\n", string(checksum), string(b))

	// 	st = st.Append(state.Entry{
	// 		Name: dest,
	// 		File: &opfs.StateFileEntry{
	// 			SHA256: checksum,
	// 		},
	// 	})
	// }
	// return st, nil
}

func (op Template) Run(octx operator.Context) error {
	opts := op.Args.(*TemplateOpts)
	userData, err := readUserData(octx, opts)
	if err != nil {
		return err
	}
	for i, dest := range opts.Dests {
		b, err := executeTemplate(octx, opts.Path, dest, i, userData)
		if err != nil {
			return err
		}

		if err := os.WriteFile(octx.FS.Join(dest), b, 0644); err != nil {
			return err
		}
	}
	return nil
}

func templateArgs(cmd *cobra.Command, args []string, target interface{}) error {
	t := target.(*TemplateOpts)
	t.Path = args[0]
	t.Dests = args[1:]
	return nil
}

func readUserData(octx operator.Context, opts *TemplateOpts) (map[string]interface{}, error) {
	dataPaths, err := octx.PlanDir.Resolve("vars", opts.DataPaths)
	if err != nil {
		return nil, err
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
		return nil, err
	}
	return userData, nil
}

func executeTemplate(octx operator.Context, p string, dest string, destIdx int, userData map[string]interface{}) ([]byte, error) {
	buf := &bytes.Buffer{}
	data := templates.Data{
		Data:    userData,
		Dest:    dest,
		DestIdx: destIdx,
	}
	if err := octx.Templates.ExecuteForOp(buf, p, data); err != nil {
		return nil, err
	}
	b := buf.Bytes()
	return b, nil
}
