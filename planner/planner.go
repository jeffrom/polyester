// Package planner traverses file systems, loading state according to modules,
// managing execution, and interfacing with the command line.
package planner

import (
	"bytes"
	"context"
	"io/fs"
	"os"
	"path/filepath"

	"github.com/jeffrom/polyester/planner/shell"
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
