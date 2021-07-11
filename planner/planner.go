// Package planner traverses file systems, loading state according to modules,
// managing execution, and interfacing with the command line.
package planner

import (
	"bytes"
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"sync"

	"github.com/jeffrom/polyester/operator"
	"github.com/jeffrom/polyester/planner/shell"
)

var (
	allOps      map[string]operator.Interface
	allOptsOnce = sync.Once{}
)

func setupAllOps() {
	allOps = make(map[string]operator.Interface)
	for _, op := range Operators() {
		allOps[op.Info().Name()] = op
	}
}

type Planner struct {
	rootDir  string
	planFile string
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

func (r *Planner) Check(ctx context.Context) error {
	allOptsOnce.Do(setupAllOps)

	pf := r.getPlanFile()
	pb, err := fs.ReadFile(os.DirFS(r.rootDir), pf)
	if err != nil {
		return err
	}

	psh, err := shell.Parse(bytes.NewReader(pb))
	if err != nil {
		return err
	}
	if err := psh.Compile(ctx); err != nil {
		return err
	}
	return nil
}

func (r *Planner) getPlanFile() string {
	pf := r.planFile
	if pf == "" {
		pf = "polyester.sh"
	}
	return pf
}

type Result struct {
}
