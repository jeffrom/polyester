package compiler

import "fmt"

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

	idx := -1
	for i, plan := range resolved {
		if plan.Name == "main" {
			idx = i
			break
		}
	}
	if idx >= 0 {
		main := resolved[idx]
		resolved = append(resolved[:idx], resolved[idx+1:]...)
		resolved = append(resolved, main)
	}
	return resolved, nil
}
