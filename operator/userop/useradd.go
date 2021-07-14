package userop

import (
	"github.com/spf13/cobra"

	"github.com/jeffrom/polyester/operator"
)

type UseraddOpts struct {
	User            string `json:"user"`
	Shell           string `json:"shell,omitempty"`
	CreateHomeDir   string `json:"home_dir,omitempty"`
	Comment         string `json:"comment,omitempty"`
	CreateHome      bool   `json:"create_home,omitempty"`
	SystemUser      bool   `json:"system,omitempty"`
	CreateUserGroup bool   `json:"user_group,omitempty"`
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

	return &operator.InfoData{
		OpName: "useradd",
		Command: &operator.Command{
			Command:   cmd,
			ApplyArgs: useraddArgs,
			Target:    opts,
		},
	}
}

func (op Useradd) GetState(octx operator.Context) (operator.State, error) {
	st := operator.State{}
	return st, nil
}

func (op Useradd) Run(octx operator.Context) error {
	return nil
}

func useraddArgs(cmd *cobra.Command, args []string, target interface{}) error {
	t := target.(*UseraddOpts)
	t.User = args[0]
	return nil
}
