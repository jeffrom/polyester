package planner

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"

	"github.com/jeffrom/polyester/operator"
)

type Plan struct {
	Name         string               `json:"name"`
	Operations   []operator.Interface `json:"operations"`
	Plans        []*Plan              `json:"plans,omitempty"`
	Dependencies []*Plan              `json:"dependencies,omitempty"`
}

func ReadFile(p string) (*Plan, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	_, planName := filepath.Split(p)
	planName = strings.SplitN(planName, ".", 2)[0]

	var ops []operator.Interface
	buf := &bytes.Buffer{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Bytes()
		isSep := bytes.Equal(bytes.TrimSpace(line), []byte("---"))
		if isSep {
			op, err := opFromBuf(buf)
			if err != nil {
				return nil, fmt.Errorf("failed to extract operation: %w", err)
			}
			// fmt.Printf("after %+v\n", op.Info().Data().Command.Target)
			if op != nil {
				ops = append(ops, op)
			}
			continue
		}

		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		if _, err := buf.Write(line); err != nil {
			return nil, err
		}
		if _, err := buf.Write([]byte("\n")); err != nil {
			return nil, err
		}
		// fmt.Printf("raw line (total: %d): %s\n", buf.Len(), string(line))
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	op, err := opFromBuf(buf)
	if err != nil {
		return nil, fmt.Errorf("failed to extract last operator: %w", err)
	}
	if op != nil {
		ops = append(ops, op)
	}
	return &Plan{Name: planName, Operations: ops}, nil
}

// All returns all plans, sorted with dependencies resolved.
func (p Plan) All() ([]*Plan, error) {
	seen := make(map[string]bool)
	_, all := allPlans(&p, seen)

	sorted, err := sortPlans(all)
	if err == nil && len(sorted) != len(all) {
		panic("sorted plans were not the same length")
	}
	return sorted, err
}

func allPlans(p *Plan, seen map[string]bool) (map[string]bool, []*Plan) {
	var plans []*Plan
	for _, sp := range p.Dependencies {
		var next []*Plan
		seen, next = allPlans(sp, seen)
		plans = append(plans, next...)
	}
	for _, sp := range p.Plans {
		var next []*Plan
		seen, next = allPlans(sp, seen)
		plans = append(plans, next...)
	}
	if !seen[p.Name] {
		plans = append(plans, p)
		seen[p.Name] = true
	}
	return seen, plans
}

func sortPlans(plans []*Plan) ([]*Plan, error) {
	m := make(map[string]*Plan)
	deps := make(map[string]map[string]*Plan)
	for _, sp := range plans {
		m[sp.Name] = sp
		depm := make(map[string]*Plan)
		for _, dep := range sp.Dependencies {
			depm[dep.Name] = dep
		}
		deps[sp.Name] = depm
	}

	var resolved []*Plan
	for len(deps) > 0 {
		ready := make(map[string]bool)
		for depName, depm := range deps {
			if len(depm) == 0 {
				ready[depName] = true
			}
		}

		// circular dependency
		if len(ready) == 0 {
			var circs []string
			for name := range deps {
				circs = append(circs, name)
			}
			return nil, fmt.Errorf("circular dependency: %v", circs)
		}

		for name := range ready {
			delete(deps, name)
			resolved = append(resolved, m[name])
		}

		for name, depm := range deps {
			// reset depm to the diff w/ ready set
			diffm := make(map[string]*Plan)
			for depName, dep := range depm {
				if ready[depName] {
					continue
				}
				diffm[depName] = dep
			}
			deps[name] = diffm
		}
	}
	return resolved, nil
}

func (p Plan) TextSummary(w io.Writer) error {
	bw := bufio.NewWriter(w)
	bw.WriteString(fmt.Sprintf("plan (%d ops):\n", len(p.Operations)))
	for _, op := range p.Operations {
		fmt.Fprintf(bw, "  %s: %+v\n", op.Info().Name(), op.Info().Data().Command.Target)
	}
	return bw.Flush()
}

func opFromBuf(buf *bytes.Buffer) (operator.Interface, error) {
	defer buf.Reset()
	b := buf.Bytes()
	if len(b) == 0 {
		return nil, nil
	}
	// fmt.Printf("omg %s\n", string(b))
	entry := &operator.PlanEntry{}
	if err := yaml.Unmarshal(b, entry); err != nil {
		return nil, fmt.Errorf("failed to unmarshal operation entry: %w", err)
	}
	// fmt.Printf("omg %+v\n", string(entry.Args))
	opc, ok := allOps[entry.Name]
	if !ok {
		return nil, fmt.Errorf("did not find operation %q", entry.Name)
	}
	op := opc()
	opData := op.Info().Data()
	// fmt.Printf("buf: %p %+v\n", opData.Command.Target, opData.Command.Target)
	if err := yaml.Unmarshal(entry.Args, opData.Command.Target); err != nil {
		return nil, fmt.Errorf("failed to unmarshal operation target: %w", err)
	}
	// fmt.Printf("after: %p %+v\n", opData.Command.Target, opData.Command.Target)
	return operation{op: op, data: opData}, nil
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
