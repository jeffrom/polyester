package planner

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/jeffrom/polyester/testenv"
)

func TestSH(t *testing.T) {
	testenv.RequireEnv(t, "TESTBIN")

	t.Run("simple", testOpSHSimple)
}

func testOpSHSimple(t *testing.T) {
	tmpdir := testenv.TempPlanDir(t, testenv.Path("testdata", "shell"))
	defer testenv.RemoveOnSuccess(t, tmpdir)

	pl := newPlanner(t, filepath.Join(tmpdir, "manifest"))
	if pl == nil {
		t.Fatal("expected planner not to be nil")
	}

	opts := ApplyOpts{
		DirRoot:  filepath.Join(tmpdir, "dir"),
		StateDir: filepath.Join(tmpdir, "state"),
	}
	ctx := context.Background()
	doApply(ctx, t, pl, opts, true)
}
