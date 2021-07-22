package testenv

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/otiai10/copy"
)

func TempPlanDir(t testing.TB, fixtureDir string) string {
	t.Helper()
	if info, err := os.Stat(fixtureDir); err != nil {
		panic(err)
	} else if !info.IsDir() {
		panic(fixtureDir + " is not a directory")
	}
	tmpDir := TempDir(t, "")
	// die(os.MkdirAll(filepath.Join(tmpDir, "state"), 0755))
	// die(os.MkdirAll(filepath.Join(tmpDir, "sandbox"), 0755))
	die(copy.Copy(fixtureDir, filepath.Join(tmpDir, "manifest"), copy.Options{
		OnDirExists: func(src, dest string) copy.DirExistsAction { return copy.Replace },
	}))
	return tmpDir
}
