package planner

import (
	"bytes"
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
	Dryrun       bool
	CompiledPlan string
	DirRoot      string
	StateDir     string
}

func (o ApplyOpts) WithDefaults() ApplyOpts {
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

func (r *Planner) Apply(ctx context.Context, opts ApplyOpts) (*Result, error) {
	opts = opts.WithDefaults()
	if err := os.MkdirAll(opts.StateDir, 0700); err != nil {
		return nil, err
	}
	pfPath := r.getPlanFile()
	pb, err := fs.ReadFile(os.DirFS(r.rootDir), pfPath)
	if err != nil {
		return nil, err
	}
	planDir, err := r.resolvePlanDir(ctx)
	if err != nil {
		return nil, err
	}
	wd, err := os.Getwd()
	if err != nil {
		return nil, err
	}
	fmt.Printf("current directory: %s\nplan directory: %s\ncompiling plan: %s\n",
		wd,
		strings.TrimPrefix(planDir, wd+"/"),
		strings.TrimPrefix(filepath.Join(r.rootDir, pfPath), wd+"/"))

	tmpDir, err := r.getIntermediatePlanDir(opts)
	if err != nil {
		return nil, err
	}
	plan, err := r.compileMainPlan(ctx, tmpDir, pb)
	if err != nil {
		return nil, err
	}
	fmt.Println("temp dir:", tmpDir)

	if err := r.checkPlan(ctx, plan, tmpDir, opts); err != nil {
		return nil, err
	}

	stateDir, err := r.setupState(plan, opts)
	if err != nil {
		return nil, err
	}
	res, err := r.executePlans(ctx, plan, stateDir, opts)
	if err != nil {
		return nil, err
	}
	if err := r.pruneState(plan, stateDir); err != nil {
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

func (r *Planner) getIntermediatePlanDir(opts ApplyOpts) (string, error) {
	tmpDir, err := ioutil.TempDir("", "polyester")
	if err != nil {
		return "", err
	}
	return tmpDir, nil
}

// compileMainPlan writes a single plan, and any of its dependencies, into the
// local filesystem.
func (r *Planner) compileMainPlan(ctx context.Context, tmpDir string, planb []byte) (*Plan, error) {
	if err := r.execPlanDeclaration(ctx, tmpDir, "plan", planb); err != nil {
		return nil, err
	}
	plan, err := r.resolvePlan(ctx, tmpDir)
	return plan, err
}

func (r *Planner) execPlanDeclaration(ctx context.Context, dir, name string, planb []byte) error {
	if err := os.MkdirAll(filepath.Join(dir, "_scripts"), 0700); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(dir, "plans"), 0700); err != nil {
		return err
	}
	scriptFile := filepath.Join(dir, "_scripts", name+".sh")
	if err := ioutil.WriteFile(scriptFile, annotatePlanScript(planb), 0700); err != nil {
		return err
	}

	if err := r.validateScript(scriptFile); err != nil {
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
	environ, err := addSelfPathToEnviron(environ)
	if err != nil {
		return err
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
	plan, err := ReadPlan(filepath.Join(dir, "plan.yaml"))
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
		if len(planNames) > 0 {
			fmt.Printf("resolving plan(s): %v\n", planNames)
		}
		for _, planName := range planNames {
			planb, err := os.ReadFile(filepath.Join(r.planDir, "plans", planName, "install.sh"))
			if err != nil {
				return fmt.Errorf("failed to read plan file: %w", err)
			}
			if err := r.execPlanDeclaration(ctx, dir, planName, planb); err != nil {
				return fmt.Errorf("failed to compile plan %q: %w", planName, err)
			}

			subplan, err := ReadPlan(filepath.Join(dir, "plans", planName+".yaml"))
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

func (r *Planner) executePlans(ctx context.Context, plan *Plan, stateDir string, opts ApplyOpts) (*Result, error) {
	dirRoot := opts.DirRoot
	all, err := plan.All()
	if err != nil {
		return nil, err
	}

	octx := operator.NewContext(ctx, opfs.New(dirRoot), opfs.NewPlanDirFS(r.planDir))
	finalRes := &Result{}
	for _, subplan := range all {
		// fmt.Println("executeManifest", subplan.Name)
		// TODO collect failures but run all plans, and report at the end
		// (unless --fail-fast).
		res, err := r.executePlan(octx, subplan, stateDir, opts)
		if err != nil {
			return finalRes, err
		}
		if res != nil {
			finalRes.Plans = append(finalRes.Plans, res)
		}
	}

	return finalRes, nil
}

// executePlan runs a single plan
func (r *Planner) executePlan(octx operator.Context, plan *Plan, stateDir string, opts ApplyOpts) (*PlanResult, error) {
	if plan.Name != "plan" {
		// fmt.Println("woop", r.planDir, plan.Name)
		octx = octx.WithSubplan(filepath.Join(r.planDir, "plans", plan.Name))
	}
	prevs, currs, err := r.readOpStates(octx, plan, stateDir, opts)
	if err != nil {
		return nil, err
	}
	if err := plan.TextSummary(os.Stdout, prevs, currs); err != nil {
		return nil, err
	}
	dirty := false
	finalRes := &PlanResult{Name: plan.Name}
	for i, op := range plan.Operations {
		res, err := r.executeOperation(octx, op, stateDir, opts, dirty, prevs[i], currs[i])
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

func (r *Planner) readOpStates(octx operator.Context, plan *Plan, stateDir string, opts ApplyOpts) ([]operator.State, []operator.State, error) {
	var prevs []operator.State
	var currs []operator.State
	for _, op := range plan.Operations {
		prev, curr, err := r.readOpState(octx, op, stateDir, opts)
		if err != nil {
			return nil, nil, err
		}
		prevs = append(prevs, prev)
		currs = append(currs, curr)
	}
	return prevs, currs, nil
}

func (r *Planner) readOpState(octx operator.Context, op operator.Interface, stateDir string, opts ApplyOpts) (operator.State, operator.State, error) {
	info := op.Info()
	name := info.Name()

	// skip planops because planner handles running them outside this context
	if name == "plan" || name == "dependency" {
		return operator.State{}, operator.State{}, nil
	}

	data := info.Data()
	prevst, err := readPrevState(data, stateDir)
	if err != nil {
		return prevst, operator.State{}, err
	}
	st, err := op.GetState(octx)
	if err != nil {
		return prevst, st, err
	}
	return prevst, st, nil
}

func (r *Planner) executeOperation(octx operator.Context, op operator.Interface, stateDir string, opts ApplyOpts, dirty bool, prevst, st operator.State) (*OperationResult, error) {
	prevDirty := dirty
	info := op.Info()
	name := info.Name()
	// skip planops because planner handles running them outside this context
	if name == "plan" || name == "dependency" {
		return nil, nil
	}
	data := info.Data()

	res := &OperationResult{
		prevState: prevst,
		currState: st,
	}

	prevSrcSt := prevst.Source()
	srcSt := st.Source()

	prevEmpty := prevSrcSt.Empty()
	changed := prevSrcSt.Changed(srcSt)
	dirty = dirty || prevEmpty || changed
	executed := false
	if dirty {
		dryrunLabel := ""
		if opts.Dryrun {
			dryrunLabel = " (dryrun)"
		}
		fmt.Printf("-> execute %s%s (%+v)\n", op.Info().Name(), dryrunLabel, data.Command.Target)

		if !opts.Dryrun {
			executed = true
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

			targetSt := nextSt.Target()
			// fmt.Println("ASDF")
			// targetSt.WriteTo(os.Stdout)
			// fmt.Println("\n", targetSt.Changed(prevst.Target()))
			if !targetSt.Empty() && !targetSt.Changed(prevst.Target()) {
				fmt.Println("-> target state hasn't changed after execution")
				if !prevDirty {
					dirty = false
				}
			}
		}
	}

	// fmt.Printf("%25s: [empty: %8v] [changed: %8v] [dirty: %8v]\n", op.Info().Name(), prevSrcSt.Empty(), changed, dirty)
	formatOpComplete(os.Stdout, op.Info().Name(), prevSrcSt.Empty(), changed, dirty, executed)
	res.PrevEmpty = prevEmpty
	res.Changed = changed
	res.Dirty = dirty
	res.Executed = executed
	return res, nil
}

var planDeclBoilerplate = []byte(`# --- START polyester script boilerplate
alias P=polyester

# --- END polyester script boilerplate
`)

// annotatePlanScript adds boilerplate to plan script before executing them.
// All it currently adds is: alias P polyester
func annotatePlanScript(planb []byte) []byte {
	res := make([]byte, len(planb)+len(planDeclBoilerplate))
	// if the first line is a shebang, put the boilerplate on the second line
	if bytes.HasPrefix(planb, []byte("#!")) {
		idx := bytes.Index(planb, []byte("\n"))
		if idx == -1 || len(planb) < idx+1 {
			return planb
		}
		firstLine := planb[:idx+1]
		copy(res, firstLine)
		copy(res[idx+1:], planDeclBoilerplate)
		copy(res[idx+1+len(planDeclBoilerplate):], planb[idx+1:])
		return res
	}
	copy(res, planDeclBoilerplate)
	copy(res[len(planDeclBoilerplate):], planb)
	return res
}

func addSelfPathToEnviron(environ []string) ([]string, error) {
	// make sure we're using the current polyester binary for compilation
	abs, err := filepath.Abs(os.Args[0])
	if err != nil {
		return nil, err
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
		pathEnv := fmt.Sprintf("PATH=%s:/bin:/usr/bin:/usr/local/bin", selfDir)
		environ = append(environ, pathEnv)
	}
	return environ, nil
}
