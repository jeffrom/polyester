package userop

import (
	"errors"
	"os"
	"os/exec"
	"os/user"

	"github.com/mattn/go-isatty"
	"github.com/spf13/cobra"

	"github.com/jeffrom/polyester/operator"
	"github.com/jeffrom/polyester/state"
)

type UseraddOpts struct {
	User            string   `json:"user"`
	Shell           string   `json:"shell,omitempty"`
	CreateHomeDir   string   `json:"home_dir,omitempty"`
	Comment         string   `json:"comment,omitempty"`
	CreateHome      bool     `json:"create_home,omitempty"`
	SystemUser      bool     `json:"system,omitempty"`
	CreateUserGroup bool     `json:"user_group,omitempty"`
	Groups          []string `json:"groups,omitempty"`
	AddGroups       []string `json:"add_groups,omitempty"`
	RemoveGroups    []string `json:"remove_groups,omitempty"`
}

type Useradd struct {
	Args interface{}
}

func (op Useradd) Info() operator.Info {
	opts := op.Args.(*UseraddOpts)

	cmd := &cobra.Command{
		Use:   "useradd user",
		Args:  cobra.ExactArgs(1),
		Short: "adds a user",
	}
	flags := cmd.Flags()
	flags.StringVarP(&opts.Shell, "shell", "s", "", "user login `shell`")
	flags.StringVarP(&opts.CreateHomeDir, "home-dir", "d", "", "create and use `dir` for home directory")
	flags.StringVarP(&opts.Comment, "comment", "c", "", "description of user")
	flags.BoolVarP(&opts.CreateHome, "create-home", "m", false, "create home directory")
	flags.BoolVarP(&opts.SystemUser, "system", "r", false, "create a system account")
	flags.BoolVarP(&opts.CreateUserGroup, "user-group", "U", false, "create group with same name as user")
	flags.StringArrayVar(&opts.Groups, "group", nil, "exclusive list of group `name`s")
	flags.StringArrayVar(&opts.AddGroups, "add-group", nil, "ensure user is added to group `name`s")
	flags.StringArrayVar(&opts.RemoveGroups, "remove-group", nil, "ensure user removed from group `name`s")

	return &operator.InfoData{
		OpName: "useradd",
		Command: &operator.Command{
			Command:   cmd,
			ApplyArgs: useraddArgs,
			Target:    opts,
		},
	}
}

func (op Useradd) GetState(octx operator.Context) (state.State, error) {
	opts := op.Args.(*UseraddOpts)
	st := state.State{}
	// TODO would be nice be able to refer to state in Run
	u, err := Lookup(opts.User)
	if err != nil && !errors.Is(err, user.UnknownUserError(opts.User)) {
		return st, err
	}
	st = st.Append(state.Entry{
		Name: "~" + opts.User,
		KV:   u.ToMap(),
	})
	// fmt.Printf("da user: %+v\n", u.ToMap())
	// st.WriteTo(os.Stdout)
	return st, nil
}

func (op Useradd) Run(octx operator.Context) error {
	opts := op.Args.(*UseraddOpts)
	u, err := Lookup(opts.User)
	if err != nil && !errors.Is(err, user.UnknownUserError(opts.User)) {
		return err
	}
	if u == nil {
		return callUseradd(octx, opts)
	}

	if err := callUsermod(octx, u, opts); err != nil {
		return err
	}

	// NOTE can use chsh to change the user login shell
	return nil
}

func callUseradd(octx operator.Context, opts *UseraddOpts) error {
	args := []string{}
	if opts.Shell != "" {
		args = append(args, "--shell", opts.Shell)
	}
	if opts.CreateHomeDir != "" {
		args = append(args, "--home-dir", opts.CreateHomeDir)
	}
	if opts.CreateHome {
		args = append(args, "--create-home")
	}
	if opts.Comment != "" {
		args = append(args, "--comment", opts.Comment)
	}
	if opts.SystemUser {
		args = append(args, "--system")
	}
	if opts.CreateUserGroup {
		args = append(args, "--user-group")
	}
	args = append(args, opts.User)

	cmd := exec.CommandContext(octx.Context, "useradd", args...)
	if isatty.IsTerminal(os.Stdout.Fd()) {
		cmd.Stdin = os.Stdin
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func callUsermod(octx operator.Context, curr *User, opts *UseraddOpts) error {
	args := []string{}
	if curr.Shell != opts.Shell {
		args = append(args, "--shell", opts.Shell)
	}
	if curr.Name != opts.Comment {
		args = append(args, "--comment", opts.Comment)
	}
	if curr.HomeDir != opts.CreateHomeDir {
		args = append(args, "--home", opts.CreateHomeDir)
	}

	args = append(args, opts.User)

	cmd := exec.CommandContext(octx.Context, "usermod", args...)
	if isatty.IsTerminal(os.Stdout.Fd()) {
		cmd.Stdin = os.Stdin
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

func useraddArgs(cmd *cobra.Command, args []string, target interface{}) error {
	t := target.(*UseraddOpts)
	t.User = args[0]
	return nil
}
