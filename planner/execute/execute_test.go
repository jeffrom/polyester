package execute

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/jeffrom/polyester/compiler"
	"github.com/jeffrom/polyester/manifest"
	"github.com/jeffrom/polyester/operator"
	"github.com/jeffrom/polyester/operator/opfs"
	"github.com/jeffrom/polyester/testenv"
)

func TestPool(t *testing.T) {
	tcs := []struct {
		name string
		dir  string
	}{
		{
			name: "noop",
			dir:  testenv.Path("testdata", "noop"),
		},
		{
			name: "basic",
			dir:  testenv.Path("testdata", "basic"),
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			ctx := context.Background()
			tmpdir := testenv.TempPlanDir(t, tc.dir)
			defer testenv.RemoveOnSuccess(t, tmpdir)
			planDir := filepath.Join(tmpdir, "manifest")
			dirRoot := filepath.Join(tmpdir, "dir")
			stateDir := filepath.Join(tmpdir, "state")

			octx := operator.NewContext(ctx, opfs.New(dirRoot), opfs.NewPlanDirFS(planDir), nil)
			opts := Opts{
				DirRoot:  dirRoot,
				StateDir: stateDir,
			}
			ep := newExecPool(1)
			ep.start(octx, opts)

			mani, err := manifest.LoadDir(planDir)
			if err != nil {
				t.Fatalf("manifest.LoadDir failed: %+v", err)
			}
			pl, err := compiler.New().Compile(ctx, mani)
			if err != nil {
				t.Fatalf("compile failed: %+v", err)
			}

			ep.add(pl)

			res, err := ep.wait()
			if err != nil {
				t.Fatalf("execPool wait failed: %+v", err)
			}
			if res == nil {
				t.Fatal("result was nil")
			}
		})
	}
}
