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

	tmpDir, err := r.compilePlan(ctx, pb)
	if err != nil {
		return Result{}, err
	}
	fmt.Println("tmpdir:", tmpDir)

	plan, err := ReadFile(filepath.Join(tmpDir, "plan"))
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

	// prevst, err := r.readPrevState(octx, plan, opts.StateDir)
	// if err != nil {
	// 	return Result{}, err
	// }

	// st, err := r.gatherState(octx, plan)
	// if err != nil {
	// 	return Result{}, err
	// }

	// fmt.Printf("gathered state: %+v\n", st)
	// if _, err := st.WriteTo(os.Stdout); err != nil {
	// 	return Result{}, err
	// }

	if err := r.executePlan(octx, plan, opts.StateDir); err != nil {
		return Result{}, err
	}

	return Result{}, nil
}

func (r *Planner) compilePlan(ctx context.Context, planb []byte) (string, error) {
	tmpDir, err := ioutil.TempDir("", "polyester")
	if err != nil {
		return "", err
	}

	scriptFile := filepath.Join(tmpDir, "plan-script.sh")
	if err := ioutil.WriteFile(scriptFile, planb, 0700); err != nil {
		return tmpDir, err
	}

	tmpPlanFile := filepath.Join(tmpDir, "plan")
	environ := []string{
		fmt.Sprintf("_POLY_PLAN=%s", tmpPlanFile),
	}
	if _, err := exec.LookPath("polyester"); err != nil {
		abs, err := filepath.Abs(os.Args[0])
		if err != nil {
			return "", err
		}
		dir, _ := filepath.Split(abs)
		dir = filepath.Clean(dir)
		found := false
		for i, env := range environ {
			parts := strings.SplitN(env, "=", 2)
			key := parts[0]
			if key != "PATH" {
				continue
			}

			environ[i] = env + ":" + dir
			fmt.Println("set $PATH=", environ[i])
			found = true
			break
		}

		if !found {
			environ = append(environ, "PATH="+dir)
		}
	}

	cmd := exec.CommandContext(ctx, scriptFile)
	cmd.Env = append(os.Environ(), environ...)
	if isatty.IsTerminal(os.Stdout.Fd()) {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
	}

	if err := cmd.Start(); err != nil {
		return tmpDir, err
	}
	if err := cmd.Wait(); err != nil {
		return tmpDir, err
	}
	return tmpDir, nil
}

// func (r *Planner) readPrevState(octx operator.Context, op operator.Interface, stateDir string) (operator.State, error) {
// 	st := operator.State{}

// 	return st, nil
// }

// func (r *Planner) gatherState(octx operator.Context, plan *Plan) (operator.State, error) {
// 	st := operator.State{}
// 	for _, op := range plan.Operations {
// 		nextSt, err := op.GetState(octx)
// 		if err != nil {
// 			return st, err
// 		}
// 		st = st.Append(nextSt.Entries...)
// 	}
// 	return st, nil
// }

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
		fmt.Printf("%20s: [empty: %8v] [changed: %8v]\n", op.Info().Name(), prevst.Empty(), prevst.Changed(st))
		if dirty || prevEmpty || changed {
			dirty = true
			fmt.Printf("executing %s (%+v)\n", op.Info().Name(), data.Command.Target)

			if err := op.Run(octx); err != nil {
				return err
			}

			nextSt, err := op.GetState(octx)
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
