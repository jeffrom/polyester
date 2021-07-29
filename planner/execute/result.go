package execute

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"

	"github.com/jeffrom/polyester/compiler"
	"github.com/jeffrom/polyester/operator"
	"github.com/jeffrom/polyester/planner/format"
	"github.com/jeffrom/polyester/state"
)

type Result struct {
	Plans []*PlanResult `json:"plans"`
}

func (r Result) Changed() bool {
	for _, pl := range r.Plans {
		if pl.Changed {
			return true
		}
	}
	return false
}

func (r Result) TextSummary(w io.Writer) error {
	bw := bufio.NewWriter(w)
	bw.WriteString(fmt.Sprintf("%d plan(s):\n", len(r.Plans)))
	if err := r.writeStateChanges(bw); err != nil {
		return err
	}
	for _, plan := range r.Plans {
		// bw.WriteString("---\n")
		label := "dirty"
		if !plan.Changed {
			label = "clean"
		}
		bw.WriteString(fmt.Sprintf("%20s(%d): %s\n", plan.Name, len(plan.Operations), label))

		if plan.Changed {
			bw.WriteString(fmt.Sprintf("plan %s changed:\n", plan.Name))
			for _, opRes := range plan.Operations {
				if !opRes.Changed {
					continue
				}
				origOp, err := compiler.GetOperation(opRes.op)
				if err != nil {
					return err
				}
				var opFmt string
				if sr, ok := origOp.(fmt.Stringer); ok {
					opFmt = sr.String()
				} else {
					b, err := json.Marshal(origOp.Info().Data().Command.Target)
					if err != nil {
						return err
					}
					opFmt = string(b)
				}

				bw.WriteString(fmt.Sprintf("%s %s -> state change:\n", opRes.Name, opFmt))
				prevst, currst := opRes.prevState, opRes.currState
				if err := prevst.Diff(bw, currst); err != nil {
					return err
				}

				// fmt.Println(plan.Name, opRes.Name, "state diff:", dmp.DiffText2(diffs))
			}
		}
	}
	return bw.Flush()
}

func (r Result) writeStateChanges(bw *bufio.Writer) error {
	planChanges := 0
	for _, plan := range r.Plans {
		for _, opRes := range plan.Operations {
			if opRes.Changed {
				planChanges++
				break
			}
		}
	}
	if planChanges == 0 {
		return nil
	}

	bw.WriteString(fmt.Sprintf("state changes for %d plan(s):\n", planChanges))

	tw := format.NewTabWriter(bw)
	format.WriteTabHeader(tw,
		"PLAN",
		"OPERATION",
		"EMPTY",
		"PREV",
		"FINAL",
		"ARGUMENTS",
	)
	for _, plan := range r.Plans {
		for _, opRes := range plan.Operations {
			if !opRes.Changed {
				continue
			}
			origOp, err := compiler.GetOperation(opRes.op)
			if err != nil {
				return err
			}
			var chgfn func(a, b state.State) (bool, error)
			if cop, ok := origOp.(operator.ChangeDetector); ok {
				chgfn = cop.Changed
			} else {
				chgfn = func(a, b state.State) (bool, error) { return a.Changed(b), nil }
			}

			prevCurrChanged, err := chgfn(opRes.prevState, opRes.currState)
			if err != nil {
				return err
			}
			prevFinalChanged, err := chgfn(opRes.prevState, opRes.finalState)
			if err != nil {
				return err
			}

			var opFmt string
			if sr, ok := origOp.(fmt.Stringer); ok {
				opFmt = sr.String()
			} else {
				b, err := json.Marshal(origOp.Info().Data().Command.Target)
				if err != nil {
					return err
				}
				opFmt = string(b)
			}
			format.WriteTabRow(tw,
				plan.Name,
				opRes.Name,
				format.Bool(opRes.prevState.Empty()),
				format.Bool(prevCurrChanged),
				format.Bool(prevFinalChanged),
				opFmt,
			)
		}
	}
	return tw.Flush()
}

// func (r Result) writeChangeTable(bw *bufio.Writer) error {
// 	return nil
// }

type PlanResult struct {
	Name       string             `json:"name"`
	Operations []*OperationResult `json:"operations"`
	Changed    bool               `json:"changed"`
	Error      error              `json:"error"`
	Plan       *compiler.Plan
}

type OperationResult struct {
	Name      string `json:"name"`
	Dirty     bool   `json:"dirty"`
	Changed   bool   `json:"changed"`
	PrevEmpty bool   `json:"prev_empty"`
	Executed  bool   `json:"executed"`

	op         operator.Interface
	prevState  state.State
	currState  state.State
	finalState state.State
}
