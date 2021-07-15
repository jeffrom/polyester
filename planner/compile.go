package planner

import (
	"context"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/mattn/go-isatty"
)

func (r *Planner) compilePlan(ctx context.Context, p string) (*Plan, error) {
	info, err := os.Stat(p)
	if err != nil {
		return nil, err
	}

	planPath := p
	if info.IsDir() {
		planPath = filepath.Join(p, "polyester.sh")
	}

	planb, err := os.ReadFile(planPath)
	if err != nil {
		return nil, err
	}

	// make sure we're using the current polyester binary for compilation
	abs, err := filepath.Abs(os.Args[0])
	if err != nil {
		return nil, err
	}
	selfDir, _ := filepath.Split(abs)
	selfDir = filepath.Clean(selfDir)

	tmpDir, err := ioutil.TempDir("", "polyester")
	if err != nil {
		return nil, err
	}
	outPath := filepath.Join(tmpDir, "plan.yaml")

	environ := []string{
		fmt.Sprintf("_POLY_PLAN=%s", outPath),
	}
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

	cmd := exec.CommandContext(ctx, "sh", "-c", string(annotatePlanScript(planb)))
	cmd.Env = append(os.Environ(), environ...)
	if isatty.IsTerminal(os.Stdout.Fd()) {
		cmd.Stdin = os.Stdin
	}
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return nil, err
	}

	plan, err := r.resolvePlan(ctx, tmpDir)
	if err != nil {
		return nil, err
	}
	return plan, os.RemoveAll(tmpDir)
}
