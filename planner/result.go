package planner

import (
	"bufio"
	"fmt"
	"io"

	"github.com/jeffrom/polyester/operator"
)

type Result struct {
	Plans []*PlanResult `json:"plans"`
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
	}
	return bw.Flush()
}

type PlanResult struct {
	Name       string             `json:"name"`
	Operations []*OperationResult `json:"operations"`
	Changed    bool               `json:"changed"`
}

type OperationResult struct {
	Dirty     bool `json:"dirty"`
	Changed   bool `json:"changed"`
	PrevEmpty bool `json:"prev_empty"`

	prevState operator.State
	currState operator.State
	nextState operator.State
}
