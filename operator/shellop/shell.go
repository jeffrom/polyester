package shellop

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"

	"github.com/spf13/cobra"

	"github.com/jeffrom/polyester/operator"
	"github.com/jeffrom/polyester/operator/fileop"
	"github.com/jeffrom/polyester/operator/opfs"
)

type ShellOpts struct {
	Script    string   `json:"script"`
	Dir       string   `json:"dir,omitempty"`
	OnChanges []string `json:"on_change,omitempty"`
	Targets   []string `json:"targets,omitempty"`
	IgnoreREs []string `json:"ignore_re,omitempty"`
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
	// TODO need to handle source v target state. Here, we want to run the
	// operator if either no on-changes were specified, or on-change files
	// changed. BUT if we run the operator and the target state doesn't change,
	// the operation should count as clean -- ie if this operation would make
	// the plan dirty, it should not if the target didn't change.
	flags.StringArrayVar(&opts.Targets, "target", nil, "track state of target `glob`")
	flags.StringArrayVar(&opts.OnChanges, "on-change", nil, "track state of source `glob`")
	flags.StringArrayVar(&opts.IgnoreREs, "ignore", nil, "ignore files matching `regex`")

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
	opts := op.Args.(*ShellOpts)
	st := operator.State{}
	for _, changeGlob := range opts.OnChanges {
		files, err := octx.FS.Glob(changeGlob)
		if err != nil {
			return st, err
		}
		for _, fp := range files {
			ig, err := shouldIgnore(fp, opts.IgnoreREs)
			if err != nil {
				return st, err
			}
			if ig {
				continue
			}

			info, err := octx.FS.Stat(fp)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return st, err
			}
			var checksum []byte
			if err == nil && !info.IsDir() {
				var err error
				checksum, err = fileop.Checksum(octx.FS.Join(fp))
				if err != nil {
					return st, err
				}
			}

			st = st.Append(operator.StateEntry{
				Name: fp,
				File: &opfs.StateFileEntry{
					Info:   info,
					SHA256: checksum,
				},
			})
		}
	}

	for _, targ := range opts.Targets {
		files, err := octx.FS.Glob(filepath.Join(opts.Dir, targ))
		if err != nil {
			return st, err
		}
		// fmt.Println("target filez from", octx.FS.Join(opts.Dir, targ), files)

		for _, fp := range files {
			ig, err := shouldIgnore(fp, opts.IgnoreREs)
			if err != nil {
				return st, err
			}
			if ig {
				continue
			}

			info, err := octx.FS.Stat(fp)
			if err != nil && !errors.Is(err, os.ErrNotExist) {
				return st, err
			}
			var checksum []byte
			if err == nil && !info.IsDir() {
				var err error
				checksum, err = fileop.Checksum(octx.FS.Join(fp))
				if err != nil {
					return st, err
				}
			}

			st = st.Append(operator.StateEntry{
				Name: fp,
				File: &opfs.StateFileEntry{
					Info:   info,
					SHA256: checksum,
				},
				Target: true,
			})
		}
	}

	// st.WriteTo(os.Stdout)
	return st, nil
}

func (op Shell) Run(octx operator.Context) error {
	opts := op.Args.(*ShellOpts)
	cmd := exec.CommandContext(octx.Context, "sh", "-c", opts.Script)
	cmd.Dir = octx.FS.Join(opts.Dir)
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

func shouldIgnore(fp string, ignores []string) (bool, error) {
	for _, ignoreRE := range ignores {
		re, err := regexp.Compile(ignoreRE)
		if err != nil {
			return false, err
		}
		if re.MatchString(fp) {
			return true, nil
		}
	}
	return false, nil
}
