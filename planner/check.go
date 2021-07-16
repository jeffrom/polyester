package planner

import (
	"bytes"
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/jeffrom/polyester/operator"
	"github.com/jeffrom/polyester/operator/opfs"
	"github.com/jeffrom/polyester/planner/shell"
	"github.com/spf13/cobra"
)

func (r *Planner) Check(ctx context.Context) error {
	allOptsOnce.Do(setupAllOps)

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

func (r *Planner) checkPlan(ctx context.Context, plan *Plan, tmpDir string, opts ApplyOpts) error {
	if err := r.preValidate(ctx, tmpDir); err != nil {
		return err
	}
	return r.intermediateValidate(ctx, tmpDir, opts)
}

func (r *Planner) preValidate(ctx context.Context, tmpDir string) error {
	scriptRoot := filepath.Join(tmpDir, "_scripts")
	mainScriptPath := filepath.Join(scriptRoot, "plan.sh")
	if _, err := os.Stat(mainScriptPath); err != nil {
		return err
	}

	allScripts := []string{mainScriptPath}
	// TODO get rest of them

	walkFn := func(p string, d fs.FileInfo, perr error) error {
		if perr != nil {
			return perr
		}
		if d.IsDir() || d.Name() == "plan.sh" {
			return nil
		}
		allScripts = append(allScripts, p)
		return nil
	}
	if err := filepath.Walk(scriptRoot, walkFn); err != nil {
		return err
	}

	for _, scriptPath := range allScripts {
		if err := r.validateScript(scriptPath); err != nil {
			return err
		}
	}
	return nil
	// return errors.New("bork")
}

func (r *Planner) validateScript(p string) error {
	_, fileName := filepath.Split(p)
	var relPath string
	if fileName == "plan.sh" {
		relPath = "polyester.sh"
	} else {
		prefix := strings.SplitN(fileName, ".", 2)[0]
		relPath = filepath.Join("plans", prefix, "install.sh")
	}

	shb, err := os.ReadFile(p)
	if err != nil {
		return err
	}
	psh, err := shell.Parse(bytes.NewReader(shb))
	if err != nil {
		return err
	}
	stmts, err := psh.Extract()
	if err != nil {
		return err
	}

	octx := operator.NewContext(context.Background(), nil, opfs.NewPlanDirFS(r.planDir))
	for _, callExpr := range stmts {
		lits := shell.Literals(callExpr.Args)
		cmd, rawArgs := lits[1], lits[2:]
		opc, ok := allOps[cmd]
		if !ok {
			return fmt.Errorf("%s:%s-%s: unrecognized polyester operator %q", relPath, callExpr.Pos(), callExpr.End(), cmd)
		}
		// fmt.Println("nice", cmd, rawArgs, opc().Info().Name())

		op := opc()
		data := op.Info().Data()
		cobraCmd := data.Command.Command
		validater, _ := op.(operator.Validater)
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

func (r *Planner) intermediateValidate(ctx context.Context, tmpDir string, opts ApplyOpts) error {
	plan, err := ResolvePlan(filepath.Join(tmpDir, "plan.yaml"))
	if err != nil {
		return err
	}
	allPlans, err := plan.All()
	if err != nil {
		return err
	}

	octx := operator.NewContext(ctx, opfs.New(opts.DirRoot), opfs.NewPlanDirFS(r.planDir))
	for _, plan := range allPlans {
		for _, op := range plan.Operations {
			validater, ok := op.(operator.Validater)
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
