package templateop

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"os/user"
	"path/filepath"
	"strings"

	"filippo.io/age"
	"filippo.io/age/armor"
	"github.com/spf13/cobra"

	"github.com/jeffrom/polyester/operator"
	"github.com/jeffrom/polyester/operator/fileop"
	"github.com/jeffrom/polyester/operator/opfs"
	"github.com/jeffrom/polyester/operator/templates"
	"github.com/jeffrom/polyester/state"
)

type TemplateOpts struct {
	Path          string   `json:"path"`
	Dests         []string `json:"dests"`
	DataPaths     []string `json:"data,omitempty"`
	IdentityPaths []string `json:"identities,omitempty"`
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
	flags.StringArrayVarP(&opts.IdentityPaths, "age-identity", "i", nil, "path(s) to age identity `file`(s)")
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
					RawName: dest,
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
	// return state.New(), nil
	opts := op.Args.(*TemplateOpts)
	st := state.State{}

	identities, err := readIdentities(octx, opts)
	if err != nil {
		return st, err
	}
	userData, err := readUserData(octx, opts)
	if err != nil {
		return st, err
	}
	secretData, err := readSecretData(octx, identities, opts)
	if err != nil {
		return st, err
	}
	// fmt.Printf("template: DesiredState opts: %+v\ndata:%+v\nsecrets: %+v\n", opts, userData, secretData)

	for i, dest := range opts.Dests {
		b, err := executeTemplate(octx, opts.Path, dest, i, userData, secretData)
		if err != nil {
			return st, err
		}

		checksum, err := fileop.ChecksumReader(bytes.NewReader(b))
		if err != nil {
			return st, err
		}
		// fmt.Printf("checksum %x, rendered:\n%s\n", checksum, string(b))

		st = st.Append(state.Entry{
			Name: dest,
			File: &opfs.StateFileEntry{
				SHA256: checksum,
				Info: opfs.StateFileInfo{
					RawName: dest,
					SHA256:  checksum,
				},
			},
		})
	}
	return st, nil
}

func (op Template) Run(octx operator.Context) error {
	opts := op.Args.(*TemplateOpts)
	identities, err := readIdentities(octx, opts)
	if err != nil {
		return err
	}
	userData, err := readUserData(octx, opts)
	if err != nil {
		return err
	}
	secretData, err := readSecretData(octx, identities, opts)
	if err != nil {
		return err
	}
	for i, dest := range opts.Dests {
		b, err := executeTemplate(octx, opts.Path, dest, i, userData, secretData)
		if err != nil {
			return err
		}

		if fi, err := os.Stat(octx.FS.Join(dest)); err == nil && fi.IsDir() {
			return fmt.Errorf("template: dir destination not supported: %q", dest)
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

func readSecretData(octx operator.Context, ids []age.Identity, opts *TemplateOpts) (map[string][]byte, error) {
	secretPaths, err := octx.PlanDir.Resolve("secrets", []string{"**/*.age"})
	if err != nil && !errors.Is(err, opfs.ErrNotFound) {
		return nil, err
	}

	res := make(map[string][]byte)
	for _, secretPath := range secretPaths {
		// fmt.Println("planDir opening", secretPath, convertSecretPath(secretPath, octx.PlanDir.Subplan()))
		f, err := octx.PlanDir.Open(secretPath)
		if err != nil {
			return nil, err
		}
		defer f.Close()

		decrypted, err := ageDecrypt(f, ids...)
		if err != nil {
			return nil, err
		}
		f.Close()
		res[convertSecretPath(secretPath, octx.PlanDir.Subplan())] = decrypted
	}
	return res, nil
}

func readIdentities(octx operator.Context, opts *TemplateOpts) ([]age.Identity, error) {
	var res []age.Identity
	// TODO proper XDG config
	homeDir := os.Getenv("HOME")
	if homeDir == "" {
		currUser, err := user.Current()
		if err != nil {
			return nil, err
		}
		homeDir = filepath.Join("home", currUser.Name)
	}
	ageCfgDir := filepath.Join(homeDir, ".config", "polyester", "age")
	if _, err := os.Stat(ageCfgDir); err == nil {
		files, err := os.ReadDir(ageCfgDir)
		if err != nil {
			return nil, err
		}
		for _, fi := range files {
			if fi.IsDir() {
				continue
			}
			// fmt.Println("zorp", filepath.Join(ageCfgDir, fi.Name()))
			b, err := os.ReadFile(filepath.Join(ageCfgDir, fi.Name()))
			if err != nil {
				return nil, err
			}
			ids, err := age.ParseIdentities(bytes.NewReader(b))
			if err != nil {
				return nil, err
			}
			res = append(res, ids...)
		}
	}

	for _, p := range opts.IdentityPaths {
		b, err := octx.FS.ReadFile(p)
		if err != nil {
			return nil, err
		}
		ids, err := age.ParseIdentities(bytes.NewReader(b))
		if err != nil {
			return nil, err
		}

		res = append(res, ids...)
	}
	return res, nil
}

func executeTemplate(octx operator.Context, p string, dest string, destIdx int, userData map[string]interface{}, secretData map[string][]byte) ([]byte, error) {
	buf := &bytes.Buffer{}
	data := templates.Data{
		Data:    userData,
		Secrets: secretData,
		Dest:    dest,
		DestIdx: destIdx,
	}
	resolved, err := resolveTemplatePath(octx, p)
	if err != nil {
		return nil, err
	}
	// fmt.Printf("template %s (resolved: %s) -> %s\n", p, resolved, dest)
	if err := octx.Templates.ExecuteForOp(buf, resolved, data); err != nil {
		return nil, err
	}
	b := buf.Bytes()
	return b, nil
}

func ageDecrypt(src io.Reader, identities ...age.Identity) ([]byte, error) {
	rr := bufio.NewReader(src)
	if start, _ := rr.Peek(len(armor.Header)); string(start) == armor.Header {
		src = armor.NewReader(rr)
	} else {
		src = rr
	}
	decrypted, err := age.Decrypt(src, identities...)
	if err != nil {
		return nil, err
	}

	return io.ReadAll(decrypted)
}

const sep = string(filepath.Separator)

func convertSecretPath(p, spdir string) string {
	if strings.HasPrefix(p, "secrets"+sep) {
		p = strings.TrimPrefix(p, "secrets"+sep)
	}
	dir, file := filepath.Split(p)
	var convertedFile string
	switch filepath.Ext(file) {
	case ".age":
		convertedFile = strings.TrimSuffix(file, ".age")
	default:
		panic("template: secret file extension not supported: " + p)
	}
	if strings.HasPrefix(p, filepath.Join("plans", spdir)) {
		return convertedFile
	}
	// fmt.Println("convertSecretPath", convertedFile, filepath.Join(dir, convertedFile))
	return filepath.Join(dir, convertedFile)
}

func resolveTemplatePath(octx operator.Context, p string) (string, error) {
	if sp := octx.PlanDir.Subplan(); sp != "" {
		return filepath.Join(sp, p), nil
	}
	return p, nil
}
