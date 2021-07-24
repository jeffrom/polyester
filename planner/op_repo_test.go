package planner

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jeffrom/polyester/testenv"
)

func TestRepo(t *testing.T) {
	testenv.RequireEnv(t, "TESTBIN")

	t.Run("simple", testOpRepoSimple)
}

func testOpRepoSimple(t *testing.T) {
	tmpdir := testenv.TempPlanDir(t, testenv.Path("testdata", "repo"))
	defer testenv.RemoveOnSuccess(t, tmpdir)
	repoURL := filepath.Join(tmpdir, "bare.git")
	testenv.GitBare(t, repoURL)
	cloneDir := filepath.Join(tmpdir, "cloned")
	testenv.GitClone(t, repoURL, cloneDir)
	binPath := filepath.Join(cloneDir, "coolbin")
	testenv.WriteFile(t, binPath, `echo "cool"`)
	testenv.GitCommitAllPush(t, cloneDir, "initial commit")

	os.Setenv("REPO_URL", repoURL)
	defer os.Unsetenv("REPO_URL")

	ctx := context.Background()

	pl := newPlanner(t, filepath.Join(tmpdir, "manifest"))
	if pl == nil {
		t.Fatal("expected planner not to be nil")
	}

	opts := ApplyOpts{
		DirRoot:  filepath.Join(tmpdir, "dir"),
		StateDir: filepath.Join(tmpdir, "state"),
	}
	doApply(ctx, t, pl, opts, true)
	doApply(ctx, t, pl, opts, false)

	testenv.WriteFile(t, binPath, `echo "nice change"`)
	testenv.GitCommitAllPush(t, cloneDir, "change coolbin")
	doApply(ctx, t, pl, opts, true)

	coolbin := testenv.ReadFile(t, binPath)
	if !strings.Contains(coolbin, "nice change") {
		t.Fatalf(`expected "nice change" in bin but was: %q`, coolbin)
	}
	doApply(ctx, t, pl, opts, false)
}

func doApply(ctx context.Context, t testing.TB, pl *Planner, opts ApplyOpts, expectChange bool) *Result {
	t.Helper()
	if pl == nil {
		t.Fatal("expected planner not to be nil")
	}

	res, err := pl.Apply(ctx, opts)
	if err != nil {
		t.Fatal("apply failed", err)
	}
	if res == nil {
		t.Fatal("expected apply result not to be nil")
		return nil
	}

	changed := res.Changed()
	if changed {
		res.TextSummary(os.Stdout)
	}
	if changed != expectChange {
		t.Fatalf("expected change (%v), got %v", expectChange, changed)
	}
	return res
}
