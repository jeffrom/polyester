package planner

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"

	"github.com/jeffrom/polyester/state"
	"github.com/sergi/go-diff/diffmatchpatch"
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
	for _, plan := range r.Plans {
		// bw.WriteString("---\n")
		label := "dirty"
		if !plan.Changed {
			label = "clean"
		}
		bw.WriteString(fmt.Sprintf("%20s(%d): %s\n", plan.Name, len(plan.Operations), label))

		if plan.Changed {
			for _, opRes := range plan.Operations {
				if !opRes.Changed {
					continue
				}

				ab, err := json.MarshalIndent(opRes.prevState.Entries, "", "  ")
				if err != nil {
					return err
				}
				bb, err := json.MarshalIndent(opRes.currState.Entries, "", "  ")
				if err != nil {
					return err
				}

				dmp := diffmatchpatch.New()
				diffs := dmp.DiffMain(string(ab), string(bb), false)
				fmt.Println(plan.Name, opRes.Name, "state diff:", dmp.DiffText2(diffs))
			}
		}
	}
	return bw.Flush()
}

type PlanResult struct {
	Name       string             `json:"name"`
	Operations []*OperationResult `json:"operations"`
	Changed    bool               `json:"changed"`
}

type OperationResult struct {
	Name      string `json:"name"`
	Dirty     bool   `json:"dirty"`
	Changed   bool   `json:"changed"`
	PrevEmpty bool   `json:"prev_empty"`
	Executed  bool   `json:"executed"`

	prevState  state.State
	currState  state.State
	finalState state.State
}
