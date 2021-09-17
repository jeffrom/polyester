package gitop

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"regexp"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"github.com/jeffrom/polyester/operator"
	"github.com/jeffrom/polyester/state"
	"github.com/jeffrom/polyester/stdio"
)

type RepoOpts struct {
	URL     string `json:"url"`
	Dest    string `json:"dest"`
	Ref     string `json:"ref,omitempty"`
	Version string `json:"version,omitempty"`
}

type Repo struct {
	Args interface{}
}

func (op Repo) Info() operator.Info {
	opts := op.Args.(*RepoOpts)
	cmd := &cobra.Command{
		Use:  "git-repo url dest",
		Args: cobra.ExactArgs(2),
	}

	flags := cmd.Flags()
	flags.StringVar(&opts.Ref, "ref", "", "the tracking ref to use (default: HEAD)")
	flags.StringVar(&opts.Version, "version", "", "The release version for the repository")

	return &operator.InfoData{
		OpName: "git-repo",
		Command: &operator.Command{
			Command:   cmd,
			ApplyArgs: repoArgs,
			Target:    opts,
		},
	}
}

func (op Repo) GetState(octx operator.Context) (state.State, error) {
	std := stdio.FromContext(octx.Context)
	opts := op.Args.(*RepoOpts)
	st := state.State{}
	std.Debugf("git-repo: GetState opts: %+v\n", opts)

	currRefStr, ref, err := getCurrentCommit(octx, octx.FS.Join(opts.Dest))
	if err != nil || currRefStr == "" {
		return st, err
	}
	gitst := &gitState{
		LocalID: currRefStr,
		Version: opts.Version,
	}

	if opts.Ref == "" || opts.Ref == "HEAD" ||
		(opts.Version != "" && opts.Version != currRefStr) {

		cmd := exec.CommandContext(octx.Context, "git", "fetch", "-v")
		cmd.Dir = octx.FS.Join(opts.Dest)
		if isatty.IsTerminal(os.Stdout.Fd()) {
			cmd.Stdin = os.Stdin
		}
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		std.Debug("+ git", cmd.Args)
		if err := cmd.Run(); err != nil {
			return st, err
		}

		remoteHead, err := getLatestCommit(octx, octx.FS.Join(opts.Dest), ref)
		if err != nil {
			return st, err
		}
		gitst.RemoteHeadID = remoteHead
	}

	st, err = st.AppendKV("git", gitst)
	return st, err
}

func (op Repo) Run(octx operator.Context) error {
	std := stdio.FromContext(octx.Context)
	opts := op.Args.(*RepoOpts)
	ctx := octx.Context

	destExists := true
	if _, err := octx.FS.Stat(opts.Dest); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		destExists = false
	}

	currRef, ref, err := getCurrentCommit(octx, octx.FS.Join(opts.Dest))
	if err != nil {
		return err
	}

	// already did fetch if the repo existed when GetState ran
	if !destExists {
		args := []string{
			"clone", opts.URL, octx.FS.Join(opts.Dest),
		}
		if opts.Ref != "" {
			args = append(args, "--branch", opts.Ref)
		}
		cmd := exec.CommandContext(ctx, "git", args...)
		std.Info("+", cmd.Args)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
	} else if opts.Version != "" && opts.Version != currRef {
		cmd := exec.CommandContext(ctx, "git", "checkout", opts.Version)
		std.Info("+", cmd.Args)
		cmd.Dir = octx.FS.Join(opts.Dest)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		if err := cmd.Run(); err != nil {
			return err
		}
	} else if opts.Ref == "" || opts.Ref == "HEAD" {
		latestRef, err := getLatestCommit(octx, octx.FS.Join(opts.Dest), ref)
		if err != nil {
			return err
		}
		if currRef != latestRef {
			cmd := exec.CommandContext(ctx, "git", "-c", "advice.detachedHead=false", "checkout", latestRef)
			std.Info("+", cmd.Args)
			cmd.Dir = octx.FS.Join(opts.Dest)
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
			if err := cmd.Run(); err != nil {
				return err
			}
		}
	}

	return nil
}

func repoArgs(cmd *cobra.Command, args []string, target interface{}) error {
	t := target.(*RepoOpts)
	t.URL = args[0]
	t.Dest = args[1]
	return nil
}

var headRefRE = regexp.MustCompile(`^ref: *(?P<ref>.*)$`)

func getHeadRef(b []byte) string {
	m := headRefRE.FindSubmatch(bytes.TrimSpace(b))
	if m == nil {
		return ""
	}
	return string(m[1])
}
