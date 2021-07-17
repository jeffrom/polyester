package pkgop

import (
	"bufio"
	"bytes"
	"os"
	"os/exec"
	"strings"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"github.com/jeffrom/polyester/operator"
	"github.com/jeffrom/polyester/state"
)

type AptInstallOpts struct {
	Packages []string
}

type AptInstall struct {
	Args interface{}
}

func (op AptInstall) Info() operator.Info {
	opts := op.Args.(*AptInstallOpts)

	cmd := &cobra.Command{
		Use:   "apt-install package...",
		Args:  cobra.MinimumNArgs(1),
		Short: "installs packages using the apt package manager",
	}
	// flags := cmd.Flags()
	// flags.Uint32VarP(&opts.Mode, "mode", "m", 0644, "the mode to set the file to")

	return &operator.InfoData{
		OpName: "apt-install",
		Command: &operator.Command{
			Command:   cmd,
			ApplyArgs: aptInstallArgs,
			Target:    opts,
		},
	}
}

func (op AptInstall) GetState(octx operator.Context) (state.State, error) {
	opts := op.Args.(*AptInstallOpts)
	st := state.State{}
	args := append([]string{"-f", "${binary:Package}@${Version}\n", "-W"}, opts.Packages...)
	cmd := exec.CommandContext(octx.Context, "dpkg-query", args...)
	outb := &bytes.Buffer{}
	cmd.Stdout = outb
	if isatty.IsTerminal(os.Stdout.Fd()) {
		cmd.Stdin = os.Stdin
	}
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return st, err
	}

	sc := bufio.NewScanner(outb)
	installed := make(map[string]interface{})
	for sc.Scan() {
		line := sc.Text()
		parts := strings.SplitN(line, "@", 2)
		name, version := parts[0], parts[1]
		installed[name] = version
	}
	if err := sc.Err(); err != nil {
		return st, err
	}
	st = st.Append(state.Entry{
		Name: "installed",
		KV:   installed,
	})

	requested := make(map[string]interface{})
	for _, arg := range opts.Packages {
		parts := strings.SplitN(arg, "@", 2)
		var version string
		name := parts[0]
		if len(parts) > 1 {
			version = parts[1]
		}
		requested[name] = version
	}
	st = st.Append(state.Entry{
		Name: "requested",
		KV:   requested,
	})
	return st, nil
}

func (op AptInstall) Run(octx operator.Context) error {
	opts := op.Args.(*AptInstallOpts)
	args := append([]string{"install", "--quiet", "--yes"}, opts.Packages...)
	cmd := exec.CommandContext(octx.Context, "apt", args...)
	if isatty.IsTerminal(os.Stdout.Fd()) {
		cmd.Stdin = os.Stdin
	}
	cmd.Stderr = os.Stderr
	cmd.Stdout = os.Stdout
	return cmd.Run()
}

func aptInstallArgs(cmd *cobra.Command, args []string, target interface{}) error {
	t := target.(*AptInstallOpts)
	t.Packages = args
	return nil
}
