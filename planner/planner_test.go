package planner

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/jeffrom/polyester/testenv"
)

// TestFakePlanCompiler is not a real test, just a helper process for other
// tests that mocks out polyester in scripts' $PATHs.
func TestFakePlanCompiler(t *testing.T) { testenv.FakePlanCompiler() }

func TestPlanner(t *testing.T) {
	testenv.RequireEnv(t, "TESTBIN")

	t.Run("noop", testNoop)
}

func testNoop(t *testing.T) {
	tmpdir := testenv.TempPlanDir(t, testenv.Path("testdata", "noop"))
	defer testenv.RemoveOnSuccess(t, tmpdir)

	pl := newPlanner(t, filepath.Join(tmpdir, "manifest"))
	if pl == nil {
		t.Fatal("expected planner not to be nil")
	}

	ctx := context.Background()
	if err := pl.Check(ctx); err != nil {
		t.Fatal("check failed", err)
	}

	opts := ApplyOpts{
		DirRoot:  filepath.Join(tmpdir, "dir"),
		StateDir: filepath.Join(tmpdir, "state"),
	}
	for i := 0; i < 3; i++ {
		res, err := pl.Apply(ctx, opts)
		if err != nil {
			t.Fatal("apply failed", err)
		}
		if res == nil {
			t.Fatal("expected apply result not to be nil")
		}

		for _, planRes := range res.Plans {
			if i == 0 && !planRes.Changed {
				t.Errorf("expected plan %q to be changed on first run", planRes.Name)
			} else if i != 0 && planRes.Changed {
				t.Errorf("expected plan %q not to be changed on run #%d", planRes.Name, i+1)
			}
		}
	}
}

func newPlanner(t testing.TB, p string) *Planner {
	t.Helper()
	pl, err := New(p)
	if err != nil {
		t.Fatal(err)
	}
	return pl
}
