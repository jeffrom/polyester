package testenv

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func Path(paths ...string) string {
	dir, err := findGoMod()
	die(err)

	finalPath := filepath.Join(append([]string{dir}, paths...)...)
	return finalPath
}

var gomodPath string

func findGoMod() (string, error) {
	if gomodPath != "" {
		return gomodPath, nil
	}

	_, file, _, ok := runtime.Caller(1)
	if !ok {
		return "", errors.New("failed to get path of caller's file")
	}
	dir, _ := filepath.Split(file)

	for d := dir; d != "/"; d, _ = filepath.Split(filepath.Clean(d)) {
		gomodPath := filepath.Join(d, "go.mod")
		if _, err := os.Stat(gomodPath); err != nil {
			continue
		}
		gomodPath = d
		return d, nil
	}
	return "", errors.New("failed to find project root")
}

func TempDir(t testing.TB, pat string) string {
	t.Helper()
	tmpdir, err := os.MkdirTemp("", "polyester-test-"+pat)
	if err != nil {
		t.Fatal(err)
	}
	t.Logf("Creating tempdir for test %s: %s", t.Name(), tmpdir)
	return tmpdir
}

func RemoveOnSuccess(t testing.TB, tmpdir string) {
	// t.Helper()
	if t.Failed() {
		t.Logf("Leaving temp dir in place because test(s) failed: %s", tmpdir)
		return
	}
	if os.Getenv("KEEPTMP") != "" {
		t.Logf("Leaving temp dir in place because $KEEPTMP was set: %s", tmpdir)
		return
	}
	logError(t, os.RemoveAll(tmpdir))
}

func RequireEnv(t testing.TB, envs ...string) {
	t.Helper()
	for _, env := range envs {
		if os.Getenv(env) == "" {
			t.Fatalf("$%s is required", env)
		}
	}
}
