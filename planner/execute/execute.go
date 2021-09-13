// Package execute contains the logic to execute plans concurrently, taking
// into account dependencies and phases.
package execute

import (
	"fmt"
	"os"
	"runtime"

	"github.com/jeffrom/polyester/compiler"
	"github.com/jeffrom/polyester/operator"
	"github.com/jeffrom/polyester/planner/format"
	"github.com/jeffrom/polyester/state"
	"github.com/jeffrom/polyester/stdio"
)

type Opts struct {
	Dryrun      bool
	DirRoot     string
	StateDir    string
	Concurrency int
}

// Execute runs a manifest concurrently. Each "plan" or "dependency" operation
// runs in a concurrent pool, taking dependencies into account. Operations run
// serially per-plan.
func Execute(octx operator.Context, plan *compiler.Plan, opts Opts) (*Result, error) {
	// safety checks:
	// - plan should only run once per apply run
	// - all of a plans dependencies must run before it is run

	// at the top level we need to keep track of the order, and put things back
	// together before returning the *Result.

	if opts.Concurrency == 0 {
		opts.Concurrency = runtime.NumCPU()
	}

	ep := newExecPool(opts.Concurrency, stdio.FromContext(octx.Context))
	ep.start(octx, opts)
	defer ep.stop()
	ep.add(plan)
	return ep.wait()
}

func executePlan(octx operator.Context, std stdio.StdIO, opts Opts, plan *compiler.Plan) (*PlanResult, error) {
	if plan.Name != "main" {
		// fmt.Println("setting subdir", plan.Name)
		spdir := octx.PlanDir.Join("plans", plan.Name)
		std.Debugf("subdir for plan %s: %s", plan.Name, spdir)
		octx = octx.WithSubplan(spdir)
		std.Debug("plan dir:", octx.PlanDir.Join(""))
	}

	prevs, currs, err := readOpStates(octx, plan, opts)
	if err != nil {
		return nil, err
	}
	if err := plan.TextSummary(std.Stdout(), prevs, currs); err != nil {
		return nil, err
	}

	dirty := false
	finalRes := &PlanResult{Plan: plan, Name: plan.Name}
	for i, op := range plan.Operations {
		res, err := executeOperation(octx, op, opts, dirty, prevs[i], currs[i])
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

func readOpStates(octx operator.Context, plan *compiler.Plan, opts Opts) ([]state.State, []state.State, error) {
	var prevs []state.State
	var currs []state.State
	for _, op := range plan.Operations {
		prev, curr, err := readOpState(octx, op, opts)
		if err != nil {
			return nil, nil, err
		}
		prevs = append(prevs, prev)
		currs = append(currs, curr)
	}
	return prevs, currs, nil
}

func readOpState(octx operator.Context, op operator.Interface, opts Opts) (state.State, state.State, error) {
	info := op.Info()
	name := info.Name()

	// skip planops because planner handles running them outside this context
	if name == "plan" || name == "dependency" {
		return state.State{}, state.State{}, nil
	}

	data := info.Data()
	prevst, err := operator.ReadState(data, opts.StateDir)
	if err != nil {
		return prevst, state.State{}, err
	}

	origOp, err := compiler.GetOperation(op)
	if err != nil {
		return prevst, state.State{}, err
	}

	st, err := origOp.GetState(octx)
	if err != nil {
		return prevst, st, err
	}
	return prevst, st, nil
}

func executeOperation(octx operator.Context, op operator.Interface, opts Opts, dirty bool, prevst, st state.State) (*OperationResult, error) {
	std := stdio.FromContext(octx.Context)
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
	changed, err := getOpChanged(octx, op, prevSrcSt, srcSt, desiredSt)
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
		std.Debugf("-> execute %s%s (%+v)", opFmt, dryrunLabel, data.Command.Target)

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

			if err := operator.SaveState(data, finalSt, opts.StateDir); err != nil {
				return nil, err
			}

			targetSt := finalSt.Target()
			// fmt.Println("ASDF")
			// targetSt.WriteTo(os.Stdout)
			// fmt.Println("\n", targetSt.Changed(prevst.Target()))
			if !targetSt.Empty() && !targetSt.Changed(prevst.Target()) {
				std.Debug("-> target state hasn't changed after execution")
				if !prevDirty {
					dirty = false
				}
			}
		}
	}

	// fmt.Printf("%25s: [empty: %8v] [changed: %8v] [dirty: %8v]\n", op.Info().Name(), prevSrcSt.Empty(), changed, dirty)
	fm := &format.DefaultFormatter{}
	fm.OpComplete(os.Stdout, op.Info().Name(), prevSrcSt.Empty(), changed, dirty, executed)
	res.op = op
	res.Name = op.Info().Name()
	res.PrevEmpty = prevEmpty
	res.Changed = changed
	res.Dirty = dirty
	res.Executed = executed
	return res, nil
}

func getOpChanged(octx operator.Context, op operator.Interface, prevst, currst, desiredst state.State) (bool, error) {
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
