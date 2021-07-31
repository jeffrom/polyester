package planner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jeffrom/polyester/compiler"
	"github.com/jeffrom/polyester/manifest"
	"github.com/jeffrom/polyester/operator"
	"github.com/jeffrom/polyester/operator/opfs"
	"github.com/jeffrom/polyester/operator/templates"
	"github.com/jeffrom/polyester/planner/execute"
	"github.com/jeffrom/polyester/state"
)

type ApplyOpts struct {
	Dryrun       bool
	CompiledPlan string
	DirRoot      string
	StateDir     string
}

func (o ApplyOpts) withDefaults() ApplyOpts {
	dirRoot := o.DirRoot
	if dirRoot == "" {
		dirRoot = "/"
	}

	stateDir := o.StateDir
	if stateDir == "" {
		stateDir = "/var/lib/polyester/state"
	}

	return ApplyOpts{
		Dryrun:       o.Dryrun,
		CompiledPlan: o.CompiledPlan,
		DirRoot:      dirRoot,
		StateDir:     stateDir,
	}
}

func (r *Planner) Apply(ctx context.Context, opts ApplyOpts) (*execute.Result, error) {
	opts = opts.withDefaults()
	if err := os.MkdirAll(opts.StateDir, 0700); err != nil {
		return nil, err
	}
	pfPath := r.getPlanFile()
	planDir, err := r.resolvePlanDir(ctx)
	if err != nil {
		return nil, err
	}
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	fmt.Printf("current directory: %s\nplan directory: %s\ncompiling plan: %s\n",
		wd,
		strings.TrimPrefix(planDir, wd+"/"),
		strings.TrimPrefix(filepath.Join(r.rootDir, pfPath), wd+"/"))

	mani, err := manifest.LoadDir(planDir)
	if err != nil {
		return nil, err
	}
	plan, err := compiler.New().Compile(ctx, mani)
	if err != nil {
		return nil, err
	}

	tmpl, err := r.setupTemplates(ctx)
	if err != nil {
		return nil, err
	}

	if err := r.checkPlan(ctx, plan, tmpl, opts); err != nil {
		return nil, err
	}

	stateDir, err := r.setupState(plan, opts)
	if err != nil {
		return nil, err
	}

	res, err := r.executePlans(ctx, plan, stateDir, tmpl, opts)
	if err != nil {
		return nil, err
	}
	if err := r.pruneState(plan, stateDir); err != nil {
		return nil, err
	}

	return res, nil
}

func (r *Planner) resolvePlanDir(ctx context.Context) (string, error) {
	lastMatch := r.rootDir
	candidate, _ := filepath.Split(filepath.Clean(r.rootDir))
	for candidate != "/" && candidate != "" {
		if _, err := os.Stat(filepath.Join(candidate, "polyester.sh")); err == nil {
			lastMatch = candidate
			break
		}

		candidate, _ = filepath.Split(filepath.Clean(candidate))
	}
	r.planDir = lastMatch
	return lastMatch, nil
}

func (r *Planner) executePlans(ctx context.Context, plan *compiler.Plan, stateDir string, tmpl *templates.Templates, opts ApplyOpts) (*execute.Result, error) {
	dirRoot := opts.DirRoot
	_, err := plan.All()
	if err != nil {
		return nil, err
	}

	octx := operator.NewContext(ctx, opfs.New(dirRoot), opfs.NewPlanDirFS(r.planDir), tmpl)
	return execute.Execute(octx, plan, execute.Opts{
		Dryrun:   opts.Dryrun,
		DirRoot:  opts.DirRoot,
		StateDir: stateDir,
	})
	// // eopts := execute.Opts{
	// // 	Dryrun:    opts.Dryrun,
	// // 	DirRoot:   opts.DirRoot,
	// // 	StateDir:  stateDir,
	// // // 	Templates: tmpl,
	// // }
	// // return execute.New(plan).Do(octx, eopts)

	// finalRes := &Result{}
	// for _, subplan := range all {
	// 	// fmt.Println("starting executePlan()", subplan.Name)
	// 	// TODO collect failures but run all plans, and report at the end
	// 	// (unless --fail-fast).
	// 	res, err := r.executePlan(octx, subplan, stateDir, opts)
	// 	if err != nil {
	// 		return finalRes, err
	// 	}
	// 	if res != nil {
	// 		finalRes.Plans = append(finalRes.Plans, res)
	// 	}
	// }

	// return finalRes, nil
}

// executePlan runs a single plan
func (r *Planner) executePlan(octx operator.Context, plan *compiler.Plan, stateDir string, opts ApplyOpts) (*PlanResult, error) {
	if plan.Name != "main" {
		// fmt.Println("setting subdir", plan.Name)
		octx = octx.WithSubplan(filepath.Join(r.planDir, "plans", plan.Name))
		// fmt.Println("executePlan plan dir:", octx.PlanDir.Join("/"))
	}
	prevs, currs, err := r.readOpStates(octx, plan, stateDir, opts)
	if err != nil {
		return nil, err
	}
	if err := plan.TextSummary(os.Stdout, prevs, currs); err != nil {
		return nil, err
	}
	dirty := false
	finalRes := &PlanResult{Name: plan.Name}
	for i, op := range plan.Operations {
		res, err := r.executeOperation(octx, op, stateDir, opts, dirty, prevs[i], currs[i])
		if err != nil {
			return nil, err
		}
		if res != nil && res.Dirty {
			dirty = true
		}
		if res != nil {
			finalRes.Operations = append(finalRes.Operations, res)
		}
	}
	if len(finalRes.Operations) == 0 {
		return nil, nil
	}
	finalRes.Changed = dirty
	return finalRes, nil
}

func (r *Planner) readOpStates(octx operator.Context, plan *compiler.Plan, stateDir string, opts ApplyOpts) ([]state.State, []state.State, error) {
	var prevs []state.State
	var currs []state.State
	for _, op := range plan.Operations {
		prev, curr, err := r.readOpState(octx, op, stateDir, opts)
		if err != nil {
			return nil, nil, err
		}
		prevs = append(prevs, prev)
		currs = append(currs, curr)
	}
	return prevs, currs, nil
}

func (r *Planner) readOpState(octx operator.Context, op operator.Interface, stateDir string, opts ApplyOpts) (state.State, state.State, error) {
	info := op.Info()
	name := info.Name()

	// skip planops because planner handles running them outside this context
	if name == "plan" || name == "dependency" {
		return state.State{}, state.State{}, nil
	}

	data := info.Data()
	prevst, err := readPrevState(data, stateDir)
	if err != nil {
		return prevst, state.State{}, err
	}
	st, err := op.GetState(octx)
	if err != nil {
		return prevst, st, err
	}
	return prevst, st, nil
}

func (r *Planner) executeOperation(octx operator.Context, op operator.Interface, stateDir string, opts ApplyOpts, dirty bool, prevst, st state.State) (*OperationResult, error) {
	prevDirty := dirty
	info := op.Info()
	name := info.Name()
	// skip planops because planner handles running them outside this context
	if name == "plan" || name == "dependency" {
		return nil, nil
	}
	data := info.Data()

	res := &OperationResult{
		prevState: prevst,
		currState: st,
	}

	prevSrcSt := prevst.Source()
	srcSt := st.Source()

	desiredSt := state.New()
	origOp, err := compiler.GetOperation(op)
	if err != nil {
		return nil, err
	}
	// fmt.Println("executeOperation plan dir:", octx.PlanDir.Join("/"))
	if dop, ok := origOp.(operator.DesiredStater); ok {
		var err error
		desiredSt, err = dop.DesiredState(octx)
		if err != nil {
			return nil, err
		}
	}
	// desiredSt.WriteTo(os.Stdout)

	prevEmpty := prevSrcSt.Empty()
	changed, err := r.getOpChanged(octx, op, prevSrcSt, srcSt, desiredSt)
	if err != nil {
		return nil, err
	}
	dirty = dirty || prevEmpty || changed
	executed := false
	if dirty {
		dryrunLabel := ""
		if opts.Dryrun {
			dryrunLabel = " (dryrun)"
		}
		opFmt := origOp.Info().Name()
		if sr, ok := origOp.(fmt.Stringer); ok {
			opFmt += " " + sr.String()
		}
		fmt.Printf("-> execute %s%s (%+v)\n", opFmt, dryrunLabel, data.Command.Target)

		if !opts.Dryrun {
			executed = true
			if err := op.Run(octx); err != nil {
				return nil, err
			}

			finalSt, err := op.GetState(octx.WithGotState(true))
			if err != nil {
				return nil, err
			}
			res.finalState = finalSt

			if err := saveState(data, finalSt, stateDir); err != nil {
				return nil, err
			}

			targetSt := finalSt.Target()
			// fmt.Println("ASDF")
			// targetSt.WriteTo(os.Stdout)
			// fmt.Println("\n", targetSt.Changed(prevst.Target()))
			if !targetSt.Empty() && !targetSt.Changed(prevst.Target()) {
				fmt.Println("-> target state hasn't changed after execution")
				if !prevDirty {
					dirty = false
				}
			}
		}
	}

	// fmt.Printf("%25s: [empty: %8v] [changed: %8v] [dirty: %8v]\n", op.Info().Name(), prevSrcSt.Empty(), changed, dirty)
	formatOpComplete(os.Stdout, op.Info().Name(), prevSrcSt.Empty(), changed, dirty, executed)
	res.op = op
	res.Name = op.Info().Name()
	res.PrevEmpty = prevEmpty
	res.Changed = changed
	res.Dirty = dirty
	res.Executed = executed
	return res, nil
}

func (r *Planner) getOpChanged(octx operator.Context, op operator.Interface, prevst, currst, desiredst state.State) (bool, error) {
	origOp, err := compiler.GetOperation(op)
	if err != nil {
		return false, err
	}
	var chgfn func(a, b state.State) (bool, error)
	if cop, ok := origOp.(operator.ChangeDetector); ok {
		chgfn = cop.Changed
	} else {
		chgfn = func(a, b state.State) (bool, error) { return a.Changed(b), nil }
	}

	// ch, err := chgfn(prevst, desiredst)
	// fmt.Println("ZXCV desired empty", desiredst.Empty(), "changed", ch, err)
	// prevst.WriteTo(os.Stdout)
	// println()
	// desiredst.WriteTo(os.Stdout)
	// println()
	if !desiredst.Empty() {
		return chgfn(prevst, desiredst)
	}
	return chgfn(prevst, currst)
}
