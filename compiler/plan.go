package compiler

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"

	"github.com/ghodss/yaml"

	"github.com/jeffrom/polyester/operator"
	"github.com/jeffrom/polyester/operator/planop"
	"github.com/jeffrom/polyester/state"
)

type Plan struct {
	Name         string               `json:"name"`
	Operations   []operator.Interface `json:"operations"`
	Plans        []*Plan              `json:"plans,omitempty"`
	Dependencies []*Plan              `json:"dependencies,omitempty"`
}

func (p Plan) RealOps() []operator.Interface {
	var res []operator.Interface
	for _, op := range p.Operations {
		if name := op.Info().Name(); name == "plan" || name == "dependency" {
			continue
		}
		res = append(res, op)
	}
	return res
}

func (p Plan) All() ([]*Plan, error) {
	seen := make(map[string]bool)
	_, all := allPlans(&p, seen)

	sorted, err := sortPlans(all)
	if err == nil && len(sorted) != len(all) {
		panic("sorted plans were not the same length")
	}
	return sorted, err
}

func (p Plan) TextSummary(w io.Writer, prevs, currs []state.State) error {
	bw := bufio.NewWriter(w)
	name := p.Name
	if name == "main" {
		name = "main plan"
	}
	bw.WriteString(fmt.Sprintf("%s (%d operations):\n", name, len(p.Operations)))
	for i, op := range p.Operations {
		n := i + 1
		chgLabel := ""
		if prevs != nil && currs != nil {
			prevst, currst := prevs[i], currs[i]
			if currst.Changed(prevst) {
				chgLabel = "X"
			}
			// bw.WriteString("prev state:\n")
			// prevst.WriteTo(bw)
		}

		origOp, err := GetOperation(op)
		if err != nil {
			return err
		}

		var opFmt string
		if sr, ok := origOp.(fmt.Stringer); ok {
			opFmt = sr.String()
		} else {
			b, err := json.Marshal(op.Info().Data().Command.Target)
			if err != nil {
				return err
			}
			opFmt = string(b)
		}
		// TODO would be nice to know here if operations changed since the last run
		fmt.Fprintf(bw, "%3d) %20s: [ %1s ] %s\n", n, op.Info().Name(), chgLabel, opFmt)
	}
	return bw.Flush()
}

func readPlan(im *intermediatePlan) (*Plan, error) {
	plans := make(map[string]*Plan)
	for name, b := range im.compiled {
		plan, err := readOnePlan(name, bytes.NewReader(b))
		if err != nil {
			return nil, err
		}
		plans[plan.Name] = plan
	}

	for _, plan := range plans {
		if err := resolveOnePlan(plan, plans); err != nil {
			return nil, err
		}
	}

	main := plans["polyester.sh"]
	return &Plan{
		Name:         "main",
		Operations:   main.Operations,
		Plans:        main.Plans,
		Dependencies: main.Dependencies,
	}, nil
}

func resolveOnePlan(plan *Plan, all map[string]*Plan) error {
	var deps []*Plan
	var plans []*Plan
	for _, op := range plan.Operations {
		info := op.Info()
		name := info.Name()
		data := info.Data()
		targ := data.Command.Target
		switch name {
		case "plan":
			args := targ.(*planop.PlanOpts).Plans
			for _, arg := range args {
				plans = append(plans, all[arg])
			}
		case "dependency":
			args := targ.(*planop.DependencyOpts).Plans
			for _, arg := range args {
				deps = append(deps, all[arg])
			}
		}
	}

	plan.Dependencies = deps
	plan.Plans = plans
	return nil
}

// readOnePlan reads some intermediate plan bytes into a struct, but does not
// resolve its subplans or dependencies.
func readOnePlan(name string, r io.Reader) (*Plan, error) {
	var ops []operator.Interface
	sc := bufio.NewScanner(r)
	sc.Split(splitOp)
	for sc.Scan() {
		b := sc.Bytes()
		if bytes.Equal(bytes.TrimSpace(b), []byte("---")) {
			continue
		}
		op, err := opFromBytes(b)
		if err != nil {
			return nil, err
		}
		ops = append(ops, op)
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}

	return &Plan{
		Name:       name,
		Operations: ops,
	}, nil
}

func splitOp(data []byte, atEOF bool) (int, []byte, error) {
	if atEOF && len(data) == 0 {
		return 0, nil, nil
	}
	if i := bytes.Index(data, []byte("\n---\n")); i >= 0 {
		// We have a full terminated operation.
		return i + 1, dropCR(data[0:i]), nil
	}
	// If we're at EOF, we have a final, non-terminated operation. Return it.
	if atEOF {
		return len(data), dropCR(data), nil
	}
	// Request more data.
	return 0, nil, nil
}

// dropCR drops a terminal \r from the data.
func dropCR(data []byte) []byte {
	if len(data) > 0 && data[len(data)-1] == '\r' {
		return data[0 : len(data)-1]
	}
	return data
}

func opFromBytes(b []byte) (operator.Interface, error) {
	// fmt.Printf("opFromBytes: %s\n", string(b))
	entry := &operator.PlanEntry{}
	if err := yaml.Unmarshal(b, entry); err != nil {
		return nil, fmt.Errorf("failed to unmarshal operation entry: %w", err)
	}

	opc, ok := allOps[entry.Name]
	if !ok {
		return nil, fmt.Errorf("did not find operation %q", entry.Name)
	}
	op := opc()
	opData := op.Info().Data()
	if len(entry.Args) > 0 && opData.Command.Target != nil {
		if err := yaml.Unmarshal(entry.Args, opData.Command.Target); err != nil {
			return nil, fmt.Errorf("failed to unmarshal operation target: %w", err)
		}
	}
	return operation{op: op, data: opData}, nil
}
