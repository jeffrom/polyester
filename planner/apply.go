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
)

type ApplyOpts struct {
	Plan     string
	DirRoot  string
	StateDir string
}

func (r *Planner) Apply(ctx context.Context, opts ApplyOpts) (Result, error) {
	pb, err := fs.ReadFile(os.DirFS(r.rootDir), r.getPlanFile())
	if err != nil {
		return Result{}, err
	}

	tmpDir, err := r.compileMainPlan(ctx, pb)
	if err != nil {
		return Result{}, err
	}
	fmt.Println("tmpdir:", tmpDir)

	plan, err := ReadFile(filepath.Join(tmpDir, "plan.yaml"))
	if err != nil {
		return Result{}, err
	}

	if err := plan.TextSummary(os.Stdout); err != nil {
		return Result{}, err
	}

	dirRoot := opts.DirRoot
	if dirRoot == "" {
		dirRoot = "/"
	}

	octx := operator.NewContext(ctx, opfs.New(dirRoot))
	if err := r.executePlan(octx, plan, opts.StateDir); err != nil {
		return Result{}, err
	}

	if err := os.RemoveAll(tmpDir); err != nil {
		return Result{}, err
	}
	return Result{}, nil
}

// compileMainPlan writes a single plan, and any of its dependencies, into the
// local filesystem.
func (r *Planner) compileMainPlan(ctx context.Context, planb []byte) (string, error) {
	tmpDir, err := ioutil.TempDir("", "polyester")
	if err != nil {
		return "", err
	}

	if err := r.execPlanDeclaration(ctx, tmpDir, "plan", planb); err != nil {
		return tmpDir, err
	}
	return tmpDir, nil
}

func (r *Planner) execPlanDeclaration(ctx context.Context, dir, name string, planb []byte) error {
	if err := os.MkdirAll(filepath.Join(dir, "_scripts"), 0700); err != nil {
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
		fmt.Println("set $PATH=", environ[i])
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
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}

func (r *Planner) resolvePlan(ctx context.Context, dir string) error {
	return nil
}

func (r *Planner) executePlan(octx operator.Context, plan *Plan, stateDir string) error {
	dirty := false
	for _, op := range plan.Operations {
		data := op.Info().Data()
		prevst, err := readPrevState(data, stateDir)
		if err != nil {
			return err
		}
		st, err := op.GetState(octx)
		if err != nil {
			return err
		}

		prevEmpty := prevst.Empty()
		changed := prevst.Changed(st)
		fmt.Printf("%20s: [empty: %8v] [changed: %8v] [dirty: %8v]\n", op.Info().Name(), prevst.Empty(), prevst.Changed(st), dirty)
		if dirty || prevEmpty || changed {
			dirty = true
			fmt.Printf("executing %s (%+v)\n", op.Info().Name(), data.Command.Target)

			if err := op.Run(octx); err != nil {
				return err
			}

			nextSt, err := op.GetState(octx.WithGotState(true))
			if err != nil {
				return err
			}

			if err := saveState(data, nextSt, stateDir); err != nil {
				return err
			}
		}
	}
	return nil
}
