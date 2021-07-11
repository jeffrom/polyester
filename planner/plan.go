package planner

import (
	"context"
	"errors"
	"os"

	"github.com/spf13/cobra"

	"github.com/jeffrom/polyester/operator"
)

type Plan struct {
	Operations []*operator.Interface
}

func AppendPlan(ctx context.Context, planFile string, info operator.Info, cobraCmd *cobra.Command, args []string) error {
	cmd := info.Data().Command
	if err := cmd.ParseFlags(args); err != nil {
		return err
	}

	if cmd.ApplyArgs != nil {
		if err := cmd.ApplyArgs(cobraCmd, args, cmd.Target); err != nil {
			return err
		}
	}

	// fmt.Printf("%s: args %+v\n", info.Name(), cmd.Target)
	return appendToFile(planFile, info)
}

func appendToFile(file string, info operator.Info) error {
	if _, err := os.Stat(file); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}

	f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	return info.Data().Encode(f)
}
