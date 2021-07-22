package planner

import (
	"bytes"
	"context"
	"io/fs"
	"os"

	"github.com/jeffrom/polyester/compiler"
	"github.com/jeffrom/polyester/operator"
	"github.com/jeffrom/polyester/operator/opfs"
	"github.com/jeffrom/polyester/operator/templates"
	"github.com/jeffrom/polyester/planner/shell"
)

func (r *Planner) Check(ctx context.Context) error {
	pf := r.getPlanFile()
	pb, err := fs.ReadFile(os.DirFS(r.rootDir), pf)
	if err != nil {
		return err
	}

	psh, err := shell.Parse(bytes.NewReader(pb))
	if err != nil {
		return err
	}
	if err := psh.Compile(ctx); err != nil {
		return err
	}
	return nil
}

func (r *Planner) getPlanFile() string {
	pf := r.planFile
	if pf == "" {
		pf = "polyester.sh"
	}
	return pf
}

func (r *Planner) checkPlan(ctx context.Context, plan *compiler.Plan, tmpl *templates.Templates, opts ApplyOpts) error {
	allPlans, err := plan.All()
	if err != nil {
		return err
	}

	octx := operator.NewContext(ctx, opfs.New(opts.DirRoot), opfs.NewPlanDirFS(r.planDir), nil)
	for _, plan := range allPlans {
		for _, op := range plan.Operations {
			validater, ok := op.(operator.Validator)
			if !ok {
				continue
			}
			if err := validater.Validate(octx, op.Info().Data().Command.Target, true); err != nil {
				return err
			}
		}
	}
	return nil
}
