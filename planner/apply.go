package planner

import (
	"context"
	"os"
	"path/filepath"
	"strings"

	"github.com/jeffrom/polyester/compiler"
	"github.com/jeffrom/polyester/manifest"
	"github.com/jeffrom/polyester/operator"
	"github.com/jeffrom/polyester/operator/opfs"
	"github.com/jeffrom/polyester/operator/templates"
	"github.com/jeffrom/polyester/planner/execute"
	"github.com/jeffrom/polyester/stdio"
)

type ApplyOpts struct {
	Dryrun       bool
	CompiledPlan string
	DirRoot      string
	StateDir     string
}

func (o ApplyOpts) withDefaults() ApplyOpts {
	dirRoot := o.DirRoot
	if dirRoot == "" {
		dirRoot = "/"
	}

	stateDir := o.StateDir
	if stateDir == "" {
		stateDir = "/var/lib/polyester/state"
	}

	return ApplyOpts{
		Dryrun:       o.Dryrun,
		CompiledPlan: o.CompiledPlan,
		DirRoot:      dirRoot,
		StateDir:     stateDir,
	}
}

func (r *Planner) Apply(ctx context.Context, opts ApplyOpts) (*execute.Result, error) {
	opts = opts.withDefaults()
	if err := os.MkdirAll(opts.StateDir, 0700); err != nil {
		return nil, err
	}
	pfPath := r.getPlanFile()
	planDir, err := r.resolvePlanDir(ctx)
	if err != nil {
		return nil, err
	}
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	std := stdio.FromContext(ctx)
	std.Infof("current directory: %s\nplan directory: %s\ncompiling plan: %s",
		wd,
		strings.TrimPrefix(planDir, wd+"/"),
		strings.TrimPrefix(filepath.Join(r.rootDir, pfPath), wd+"/"))

	mani, err := manifest.LoadDir(planDir)
	if err != nil {
		return nil, err
	}
	plan, err := compiler.New().Compile(ctx, mani)
	if err != nil {
		return nil, err
	}

	tmpl, err := r.setupTemplates(ctx)
	if err != nil {
		return nil, err
	}

	if err := r.checkPlan(ctx, plan, tmpl, opts); err != nil {
		return nil, err
	}

	stateDir, err := r.setupState(plan, opts)
	if err != nil {
		return nil, err
	}

	res, err := r.executePlans(ctx, plan, stateDir, tmpl, opts)
	if err != nil {
		return nil, err
	}
	if err := r.pruneState(stdio.FromContext(ctx), plan, stateDir); err != nil {
		return nil, err
	}

	return res, nil
}

func (r *Planner) resolvePlanDir(ctx context.Context) (string, error) {
	lastMatch := r.rootDir
	candidate, _ := filepath.Split(filepath.Clean(r.rootDir))
	for candidate != "/" && candidate != "" {
		if _, err := os.Stat(filepath.Join(candidate, "polyester.sh")); err == nil {
			lastMatch = candidate
			break
		}

		candidate, _ = filepath.Split(filepath.Clean(candidate))
	}
	r.planDir = lastMatch
	return lastMatch, nil
}

func (r *Planner) executePlans(ctx context.Context, plan *compiler.Plan, stateDir string, tmpl *templates.Templates, opts ApplyOpts) (*execute.Result, error) {
	dirRoot := opts.DirRoot
	_, err := plan.All()
	if err != nil {
		return nil, err
	}

	octx := operator.NewContext(ctx, opfs.New(dirRoot), opfs.NewPlanDirFS(r.planDir), tmpl)
	return execute.Execute(octx, plan, execute.Opts{
		Dryrun:   opts.Dryrun,
		DirRoot:  opts.DirRoot,
		StateDir: stateDir,
	})
}
