package compiler

import (
	"bytes"
	"context"
	"fmt"
	"io"

	"github.com/spf13/cobra"

	"github.com/jeffrom/polyester/compiler/shell"
	"github.com/jeffrom/polyester/operator"
)

// TODO multiple validation errors

func preValidate(ctx context.Context, im *intermediatePlan) error {
	for name, b := range im.compiled {
		if err := preValidateOne(ctx, name, bytes.NewReader(b)); err != nil {
			return err
		}
	}
	return nil
}

func preValidateOne(ctx context.Context, name string, r io.Reader) error {
	psh, err := shell.Parse(r)
	if err != nil {
		return err
	}

	stmts, err := psh.Extract()
	if err != nil {
		return err
	}
	octx := operator.NewContext(ctx, nil, nil, nil)

	for _, callExpr := range stmts {
		lits := shell.Literals(callExpr.Args)
		cmd, rawArgs := lits[1], lits[2:]
		opc, ok := allOps[cmd]
		if !ok {
			return fmt.Errorf("%s:%s-%s: unrecognized polyester operator %q", name, callExpr.Pos(), callExpr.End(), cmd)
		}
		// fmt.Println("nice", cmd, rawArgs, opc().Info().Name())

		op := opc()
		data := op.Info().Data()
		cobraCmd := data.Command.Command
		validater, _ := op.(operator.Validator)
		targ := data.Command.Target
		cobraCmd.RunE = func(cmd *cobra.Command, args []string) error {
			if data.Command.ApplyArgs != nil {
				if err := data.Command.ApplyArgs(cmd, args, targ); err != nil {
					return err
				}
			}
			if validater == nil {
				return nil
			}
			return validater.Validate(octx, targ, false)
		}
		cobraCmd.SetArgs(rawArgs)
		if err := cobraCmd.Execute(); err != nil {
			return err
		}
	}
	return nil
}

// NOTE postValidate doesn't make sense in this package
// func postValidate(ctx context.Context, plan *Plan, ofs opfs.FS, pfs opfs.PlanDir) error {
// 	all, err := plan.All()
// 	if err != nil {
// 		return err
// 	}
// 	octx := operator.NewContext(ctx, ofs, pfs, nil)
// 	for _, plan := range allPlans {
// 		for _, op := range plan.Operations {
// 			validater, ok := op.(operator.Validator)
// 			if !ok {
// 				continue
// 			}
// 			if err := validater.Validate(octx, op.Info().Data().Command.Target, true); err != nil {
// 				return err
// 			}
// 		}
// 	}
// 	return nil
// }
