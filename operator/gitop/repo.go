package gitop

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"github.com/jeffrom/polyester/operator"
	"github.com/jeffrom/polyester/operator/fileop"
	"github.com/jeffrom/polyester/operator/opfs"
)

type RepoOpts struct {
	URL  string `json:"url"`
	Dest string `json:"dest"`
	Ref  string `json:"ref,omitempty"`
}

type Repo struct {
	Args         interface{}
	gotStateOnce bool
}

func (op Repo) Info() operator.Info {
	opts := op.Args.(*RepoOpts)
	cmd := &cobra.Command{
		Use:  "git-repo url dest",
		Args: cobra.ExactArgs(2),
	}

	flags := cmd.Flags()
	flags.StringVar(&opts.Ref, "ref", "", "the ref to use (default: HEAD)")

	return &operator.InfoData{
		OpName: "git-repo",
		Command: &operator.Command{
			Command:   cmd,
			ApplyArgs: repoArgs,
			Target:    opts,
		},
	}
}

func (op Repo) GetState(octx operator.Context) (operator.State, error) {
	opts := op.Args.(*RepoOpts)
	st := operator.State{}
	fmt.Printf("git-repo: GetState opts: %+v\n", opts)

	headPath := filepath.Join(opts.Dest, ".git", "HEAD")
	headInfo, err := octx.FS.Stat(headPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return st, err
	}
	headSum, err := fileop.Checksum(octx.FS.Join(headPath))
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return st, err
	}

	st = st.Append(operator.StateEntry{
		Name: headPath,
		File: &opfs.StateFileEntry{
			Info:   headInfo,
			SHA256: headSum,
		},
	})

	if headInfo == nil {
		return st, nil
	}

	b, err := octx.FS.ReadFile(headPath)
	if err != nil {
		return st, err
	}
	ref := getHeadRef(b)
	if ref == "" {
		panic("failed to get ref from .git/HEAD: " + string(b))
	}

	currRefPath := filepath.Join(opts.Dest, ".git", ref)
	currRefInfo, err := octx.FS.Stat(currRefPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return st, err
	}
	currRef, err := octx.FS.ReadFile(currRefPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return st, err
	}

	st = st.Append(operator.StateEntry{
		Name: currRefPath,
		File: &opfs.StateFileEntry{
			Info:     currRefInfo,
			Contents: currRef,
		},
	})

	if opts.Ref == "" || opts.Ref == "HEAD" {
		cmd := exec.CommandContext(octx.Context, "git", "fetch", "-v")
		cmd.Dir = octx.FS.Join(opts.Dest)
		if isatty.IsTerminal(os.Stdout.Fd()) {
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
		}
		// fmt.Println("+ git", cmd.Args)
		if err := cmd.Run(); err != nil {
			return st, err
		}

		_, refName := filepath.Split(ref)
		cmd = exec.CommandContext(octx.Context, "git", "rev-parse", "origin/"+refName)
		cmd.Dir = octx.FS.Join(opts.Dest)
		outb := &bytes.Buffer{}
		cmd.Stderr = os.Stderr
		cmd.Stdout = outb
		if err := cmd.Run(); err != nil {
			return st, err
		}

		remoteHead := strings.TrimSpace(outb.String())
		st = st.Append(operator.StateEntry{
			Name: "remote",
			KV:   map[string]string{"HEAD": remoteHead},
		})
	}
	return st, nil
}

func (op Repo) Run(octx operator.Context) error {
	opts := op.Args.(*RepoOpts)
	ctx := octx.Context

	destExists := true
	if _, err := octx.FS.Stat(opts.Dest); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
		destExists = false
	}

	// already did fetch if the repo existed when GetState ran
	if !destExists {
		args := []string{
			"clone", opts.URL, octx.FS.Join(opts.Dest),
		}
		if opts.Ref != "" {
			args = append(args, "--branch", opts.Ref)
		}
		// fmt.Println("+ git", args)
		cmd := exec.CommandContext(ctx, "git", args...)
		if isatty.IsTerminal(os.Stdout.Fd()) {
			cmd.Stdin = os.Stdin
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr
		}
		if err := cmd.Run(); err != nil {
			return err
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
