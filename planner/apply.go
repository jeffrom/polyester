package planner

import (
	"context"
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mattn/go-isatty"

	"github.com/jeffrom/polyester/operator"
	"github.com/jeffrom/polyester/operator/opfs"
	"github.com/jeffrom/polyester/operator/planop"
)

type ApplyOpts struct {
	CompiledPlan string
	DirRoot      string
	StateDir     string
}

func (r *Planner) Apply(ctx context.Context, opts ApplyOpts) (*Result, error) {
	pfPath := r.getPlanFile()
	pb, err := fs.ReadFile(os.DirFS(r.rootDir), pfPath)
	if err != nil {
		return nil, err
	}
	planDir, err := r.resolvePlanDir(ctx)
	if err != nil {
		return nil, err
	}
	fmt.Printf("plan directory: %s\ncompiling plan: %s\n", planDir, filepath.Join(r.rootDir, pfPath))

	tmpDir, plan, err := r.compileMainPlan(ctx, pb)
	if err != nil {
		return nil, err
	}
	fmt.Println("tmpdir:", tmpDir)

	if err := plan.TextSummary(os.Stdout); err != nil {
		return nil, err
	}

	dirRoot := opts.DirRoot
	if dirRoot == "" {
		dirRoot = "/"
	}

	res, err := r.executeManifest(ctx, plan, dirRoot, opts.StateDir)
	if err != nil {
		return nil, err
	}

	if err := os.RemoveAll(tmpDir); err != nil {
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

// compileMainPlan writes a single plan, and any of its dependencies, into the
// local filesystem.
func (r *Planner) compileMainPlan(ctx context.Context, planb []byte) (string, *Plan, error) {
	tmpDir, err := ioutil.TempDir("", "polyester")
	if err != nil {
		return "", nil, err
	}

	if err := r.execPlanDeclaration(ctx, tmpDir, "plan", planb); err != nil {
		return tmpDir, nil, err
	}
	plan, err := r.resolvePlan(ctx, tmpDir)
	return tmpDir, plan, err
}

func (r *Planner) execPlanDeclaration(ctx context.Context, dir, name string, planb []byte) error {
	if err := os.MkdirAll(filepath.Join(dir, "_scripts"), 0700); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(dir, "plans"), 0700); err != nil {
		return err
	}
	scriptFile := filepath.Join(dir, "_scripts", name+".sh")
	if err := ioutil.WriteFile(scriptFile, planb, 0700); err != nil {
		return err
	}

	var planFile string
	if name == "plan" {
		// main plan case
		planFile = filepath.Join(dir, "plan.yaml")
	} else {
		planFile = filepath.Join(dir, "plans", name+".yaml")
	}

	environ := []string{
		fmt.Sprintf("_POLY_PLAN=%s", planFile),
	}
	// make sure we're using the current polyester binary for compilation
	abs, err := filepath.Abs(os.Args[0])
	if err != nil {
		return err
	}
	selfDir, _ := filepath.Split(abs)
	selfDir = filepath.Clean(selfDir)
	found := false
	for i, env := range environ {
		parts := strings.SplitN(env, "=", 2)
		key := parts[0]
		if key != "PATH" {
			continue
		}

		environ[i] = selfDir + ":" + env
		// fmt.Println("set $PATH=", environ[i])
		found = true
		break
	}
	if !found {
		environ = append(environ, "PATH="+selfDir)
	}

	cmd := exec.CommandContext(ctx, scriptFile)
	cmd.Env = append(os.Environ(), environ...)
	if isatty.IsTerminal(os.Stdout.Fd()) {
		cmd.Stdin = os.Stdin
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	fmt.Printf("+ _POLY_PLAN=%s %s \n", planFile, scriptFile)
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func (r *Planner) resolvePlan(ctx context.Context, dir string) (*Plan, error) {
	plan, err := ReadFile(filepath.Join(dir, "plan.yaml"))
	if err != nil {
		return nil, err
	}

	allPlans := make(map[string]*Plan)
	if err := r.resolveOnePlan(ctx, plan, dir, allPlans); err != nil {
		return nil, err
	}
	return plan, nil
}

func (r *Planner) resolveOnePlan(ctx context.Context, plan *Plan, dir string, allPlans map[string]*Plan) error {
	if _, ok := allPlans[plan.Name]; ok {
		return nil
	}
	for _, op := range plan.Operations {
		info := op.Info()
		name := info.Name()
		// fmt.Printf("uhhh %+v\n", name)
		targ := info.Data().Command.Target
		var planNames []string
		switch name {
		case "plan":
			planNames = targ.(*planop.PlanOpts).Plans
		case "dependency":
			planNames = targ.(*planop.DependencyOpts).Plans
		}
		fmt.Printf("resolving plan(s): %v\n", planNames)
		for _, planName := range planNames {
			planb, err := os.ReadFile(filepath.Join(r.planDir, "plans", planName, "install.sh"))
			if err != nil {
				return fmt.Errorf("failed to read plan file: %w", err)
			}
			if err := r.execPlanDeclaration(ctx, dir, planName, planb); err != nil {
				return fmt.Errorf("failed to compile plan %q: %w", planName, err)
			}

			subplan, err := ReadFile(filepath.Join(dir, "plans", planName+".yaml"))
			if err != nil {
				return err
			}
			// fmt.Println("resolve YEA:", name, subplan.Name)
			switch name {
			case "plan":
				plan.Plans = append(plan.Plans, subplan)
			case "dependency":
				plan.Dependencies = append(plan.Dependencies, subplan)
			}
		}
	}

	for _, sp := range plan.Dependencies {
		if err := r.resolveOnePlan(ctx, sp, dir, allPlans); err != nil {
			return err
		}
	}
	for _, sp := range plan.Plans {
		if err := r.resolveOnePlan(ctx, sp, dir, allPlans); err != nil {
			return err
		}
	}

	allPlans[plan.Name] = plan
	return nil
}

func (r *Planner) executeManifest(ctx context.Context, plan *Plan, dirRoot, stateDir string) (*Result, error) {
	finalRes := &Result{}
	octx := operator.NewContext(ctx, opfs.New(dirRoot))
	all, err := plan.All()
	if err != nil {
		return nil, err
	}
	for _, subplan := range all {
		// fmt.Println("executeManifest", subplan.Name)
		res, err := r.executePlan(octx, subplan, stateDir)
		if err != nil {
			return finalRes, err
		}
		if res != nil {
			finalRes.Plans = append(finalRes.Plans, res)
		}
	}

	// if the main plan has any operations, run 'em last
	// res, err := r.executePlan(octx, plan, stateDir)
	// if err != nil {
	// 	return finalRes, err
	// }
	// if res != nil {
	// 	finalRes.Plans = append(finalRes.Plans, res)
	// }
	return finalRes, nil
}

// executePlan runs a single plan
func (r *Planner) executePlan(octx operator.Context, plan *Plan, stateDir string) (*PlanResult, error) {
	dirty := false
	finalRes := &PlanResult{Name: plan.Name}
	for _, op := range plan.Operations {
		res, err := r.executeOperation(octx, op, stateDir, dirty)
		if err != nil {
			return nil, err
		}
		if res != nil && res.Dirty {
			dirty = true
		}
		if res != nil {
			finalRes.Operations = append(finalRes.Operations, res)
		}
	}
	if len(finalRes.Operations) == 0 {
		return nil, nil
	}
	finalRes.Changed = dirty
	return finalRes, nil
}

func (r *Planner) executeOperation(octx operator.Context, op operator.Interface, stateDir string, dirty bool) (*OperationResult, error) {
	info := op.Info()
	name := info.Name()
	// skip planops because planner handles running them outside this context
	if name == "plan" || name == "dependency" {
		return nil, nil
	}
	res := &OperationResult{}
	data := info.Data()
	prevst, err := readPrevState(data, stateDir)
	if err != nil {
		return nil, err
	}
	res.prevState = prevst
	st, err := op.GetState(octx)
	if err != nil {
		return nil, err
	}
	res.currState = st

	prevEmpty := prevst.Empty()
	changed := prevst.Changed(st)
	dirty = dirty || prevEmpty || changed
	fmt.Printf("%20s: [empty: %8v] [changed: %8v] [dirty: %8v]\n", op.Info().Name(), prevst.Empty(), prevst.Changed(st), dirty)
	res.PrevEmpty = prevEmpty
	res.Changed = changed
	res.Dirty = dirty
	if dirty {
		fmt.Printf("-> execute %s (%+v)\n", op.Info().Name(), data.Command.Target)

		if err := op.Run(octx); err != nil {
			return nil, err
		}

		nextSt, err := op.GetState(octx.WithGotState(true))
		if err != nil {
			return nil, err
		}
		res.nextState = nextSt

		if err := saveState(data, nextSt, stateDir); err != nil {
			return nil, err
		}
	}
	return res, nil
}
