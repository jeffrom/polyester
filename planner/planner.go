// Package planner traverses file systems, loading state according to modules,
// managing execution, and interfacing with the command line.
package planner

import "context"

type Planner struct {
	rootDir string
}

func New(dir string) *Planner {
	return &Planner{rootDir: dir}
}

func (r *Planner) Reconcile(ctx context.Context) (Result, error) {
	return Result{}, nil
}

type Result struct {
}
