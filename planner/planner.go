// Package planner traverses file systems, loading state according to modules,
// managing execution, and interfacing with the command line.
package planner

import (
	"os"
	"path/filepath"
)

type Planner struct {
	rootDir  string
	planFile string
	planDir  string
}

func New(p string) (*Planner, error) {
	// read the plan from polyester.sh. p might be a directory containing
	// polyester.sh. If p is a file, use that.

	abs, err := filepath.Abs(p)
	if err != nil {
		return nil, err
	}
	stat, err := os.Stat(abs)
	if err != nil {
		return nil, err
	}
	if stat.IsDir() {
		return &Planner{rootDir: abs}, nil
	}

	dir, file := filepath.Split(abs)
	return &Planner{rootDir: dir, planFile: file}, nil
}
