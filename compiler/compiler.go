// Package compiler contains code to compile manifests into executable plans.
//
// Only POSIX shell and bash are currently supported.
package compiler

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/jeffrom/polyester/executil"
	"github.com/jeffrom/polyester/manifest"
)

type Compiler struct {
	selfFile string
	environ  []string
}

func New() *Compiler {
	return &Compiler{}
}

func (c *Compiler) Compile(ctx context.Context, m *manifest.Manifest) (*Plan, error) {
	allOptsOnce.Do(setupAllOps)
	var environ []string
	var selfFile string
	if c.environ == nil {
		var err error
		selfFile, environ, err = addSelfPathToEnviron([]string{"_POLY_PLAN=-"})
		if err != nil {
			return nil, err
		}
		c.environ = environ
		c.selfFile = selfFile
	}

	im := newIntermediatePlan(m)
	// TODO could do this concurrently
	if err := c.execOne(ctx, im, m.Main, m.MainScript); err != nil {
		return nil, err
	}
	for name, plan := range m.Plans {
		if err := c.execOne(ctx, im, name, plan.MainScript); err != nil {
			return nil, err
		}
	}
	if err := preValidate(ctx, im); err != nil {
		return nil, err
	}

	return readPlan(im)
}

func (c *Compiler) execOne(ctx context.Context, im *intermediatePlan, name string, b []byte) error {
	script := string(annotatePlanScript(b, c.selfFile))
	// TODO statically validate script here

	r, w, err := os.Pipe()
	if err != nil {
		return err
	}
	cmd := executil.CommandContext(ctx, "sh", "-c", script)
	cmd.Env = append(os.Environ(), c.environ...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.ExtraFiles = []*os.File{w}

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("compiler: run failed: %w", err)
	}
	defer r.Close()
	w.Close()

	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, r); err != nil {
		return err
	}
	im.compiled[name] = buf.Bytes()
	return nil
}

func addSelfPathToEnviron(environ []string) (string, []string, error) {
	// make sure we're using the current polyester binary for compilation
	abs, err := filepath.Abs(os.Args[0])
	if err != nil {
		return "", nil, err
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

	// handle the unit test case here
	selfFile := abs
	if strings.HasSuffix(selfFile, ".test") {
		testEnv := os.Getenv("TESTBIN")
		selfFile = testEnv
	}
	return selfFile, environ, nil
}
