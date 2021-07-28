package planner

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/jeffrom/polyester/testenv"
)

func TestOpPcopy(t *testing.T) {
	testenv.RequireEnv(t, "TESTBIN")

	t.Run("simple", testOpPcopySimple)
	t.Run("change-source", testOpPcopyChangeSource)
}

func testOpPcopySimple(t *testing.T) {
	tmpdir := testenv.TempPlanDir(t, testenv.Path("testdata", "pcopy"))
	defer testenv.RemoveOnSuccess(t, tmpdir)

	pl := newPlanner(t, filepath.Join(tmpdir, "manifest"))
	if pl == nil {
		t.Fatal("expected planner not to be nil")
	}

	ctx := context.Background()
	if err := pl.Check(ctx); err != nil {
		t.Fatal("check failed:", err)
	}

	opts := ApplyOpts{
		DirRoot:  filepath.Join(tmpdir, "dir"),
		StateDir: filepath.Join(tmpdir, "state"),
	}
	for i := 0; i < 3; i++ {
		res, err := pl.Apply(ctx, opts)
		if err != nil {
			t.Fatal("apply failed:", err)
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

func testOpPcopyChangeSource(t *testing.T) {
	tmpdir := testenv.TempPlanDir(t, testenv.Path("testdata", "pcopy"))
	defer testenv.RemoveOnSuccess(t, tmpdir)

	ctx := context.Background()
	opts := ApplyOpts{
		DirRoot:  filepath.Join(tmpdir, "dir"),
		StateDir: filepath.Join(tmpdir, "state"),
	}
	mainPath := filepath.Join(tmpdir, "manifest", "polyester.sh")
	testenv.WriteFile(t, mainPath, `#!/bin/sh
testdir=/tmp/test/pcopy
P mkdir $testdir/a $testdir/b
P pcopy a $testdir/c
`)

	pl := newPlanner(t, filepath.Join(tmpdir, "manifest"))
	if pl == nil {
		t.Fatal("expected planner not to be nil")
	}

	res, err := pl.Apply(ctx, opts)
	if err != nil {
		t.Fatal("apply failed:", err)
	}
	if res == nil {
		t.Fatal("expected apply result not to be nil")
	}
	if !res.Changed() {
		t.Fatal("expected first run to result in a change")
	}

	testenv.WriteFile(t, mainPath, `#!/bin/sh
testdir=/tmp/test/pcopy
P mkdir $testdir/a $testdir/b
P pcopy b $testdir/c
`)
	pl = newPlanner(t, filepath.Join(tmpdir, "manifest"))
	if pl == nil {
		t.Fatal("expected planner not to be nil")
	}

	res, err = pl.Apply(ctx, opts)
	if err != nil {
		t.Fatal("apply failed", err)
	}
	if res == nil {
		t.Fatal("expected apply result not to be nil")
	}
	if !res.Changed() {
		t.Fatal("expected run with new source to result in a change")
	}

	res, err = pl.Apply(ctx, opts)
	if err != nil {
		t.Fatal("apply failed", err)
	}
	if res == nil {
		t.Fatal("expected apply result not to be nil")
	}
	if res.Changed() {
		t.Fatal("expected last run not to result in a change")
	}
}
