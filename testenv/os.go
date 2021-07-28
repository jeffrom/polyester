package testenv

import (
	"errors"
	"io/fs"
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

func Mkdirs(t testing.TB, mode fs.FileMode, dirs ...string) {
	t.Helper()
	for _, dir := range dirs {
		if err := os.MkdirAll(dir, mode); err != nil {
			t.Fatal(err)
		}
	}
}

func RequireEnv(t testing.TB, envs ...string) {
	t.Helper()
	for _, env := range envs {
		if os.Getenv(env) == "" {
			t.Fatalf("$%s is required", env)
		}
	}
}

func WriteFile(t testing.TB, p, body string) {
	t.Helper()
	if err := os.WriteFile(p, []byte(body), 0644); err != nil {
		t.Fatal("WriteFile:", err)
	}
}

func ReadFile(t testing.TB, p string) string {
	t.Helper()
	b, err := os.ReadFile(p)
	if err != nil {
		t.Fatal("ReadFile failed:", err)
	}
	return string(b)
}
