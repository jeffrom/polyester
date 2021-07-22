package manifest

import (
	"bytes"
	"crypto/sha256"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jeffrom/polyester/testenv"
)

func TestLoadDir(t *testing.T) {
	tmpdir := testenv.TempPlanDir(t, testenv.Path("testdata", "basic"))
	defer testenv.RemoveOnSuccess(t, tmpdir)

	man, err := LoadDir(filepath.Join(tmpdir, "manifest"))
	if err != nil {
		t.Fatal("LoadDir failed:", err)
	}
	if man == nil {
		t.Fatal("expected manifest not to be nil")
	}

	if len(man.Files) != 0 {
		t.Error("expected 0 root files, got", len(man.Files))
	}
	if len(man.Vars) != 0 {
		t.Error("expected 0 root vars, got", len(man.Vars))
	}
	if len(man.Secrets) != 0 {
		t.Error("expected 0 root secrets, got", len(man.Secrets))
	}
	if len(man.Templates) == 0 {
		t.Error("expected root templates not to be empty")
	}

	if man.Main != "polyester.sh" {
		t.Errorf("expected main script to be polyester.sh, was %q", man.Main)
	}
	if len(man.MainScript) == 0 {
		t.Error("expected main script not to be empty")
	}
	if len(man.Plans) == 0 {
		t.Error("expected plans not to be empty")
	}

	gitty, ok := man.Plans["gitty"]
	if !ok {
		t.Error("expected gitty plan")
	} else {
		if gitty.Main != "plan.sh" {
			t.Errorf("expected gitty main to be plan.sh, was %q", gitty.Main)
		}
		if len(gitty.MainScript) == 0 {
			t.Error("expected gitty main script not to be empty")
		}
	}
}

func TestLoadSaveDir(t *testing.T) {
	tcs := []struct {
		dir string
	}{
		{dir: testenv.Path("testdata", "apt-install")},
		{dir: testenv.Path("testdata", "atomic-copy")},
		{dir: testenv.Path("testdata", "basic")},
		{dir: testenv.Path("testdata", "copy")},
		{dir: testenv.Path("testdata", "noop")},
		{dir: testenv.Path("testdata", "pcopy")},
		{dir: testenv.Path("testdata", "shell")},
		{dir: testenv.Path("testdata", "template")},
		{dir: testenv.Path("testdata", "useradd")},
	}

	for _, tc := range tcs {
		_, name := filepath.Split(tc.dir)
		t.Run(name, func(t *testing.T) {
			tmpdir := testenv.TempPlanDir(t, tc.dir)
			defer testenv.RemoveOnSuccess(t, tmpdir)

			mandir := filepath.Join(tmpdir, "manifest")
			man, err := LoadDir(mandir)
			if err != nil {
				t.Fatal("LoadDir failed:", err)
			}
			if man == nil {
				t.Fatal("expected manifest not to be nil")
			}

			destdir := filepath.Join(tmpdir, "dest")
			if err := SaveDir(destdir, man); err != nil {
				t.Fatal("SaveDir failed:", err)
			}
			checkDirsEqual(t, mandir, destdir)
		})
	}
}

func checkDirsEqual(t testing.TB, e, a string) {
	t.Helper()
	em, am := dirToMap(e), dirToMap(a)

	if len(em) != len(am) {
		t.Errorf("expected %d files, got %d", len(em), len(am))
	}
	for k, eh := range em {
		ah, ok := am[k]
		if !ok {
			t.Errorf("missing expected file %q", k)
		} else if !bytes.Equal(eh, ah) {
			t.Errorf("file %q did not match expected value", k)
		}
	}

	for k := range am {
		if _, ok := em[k]; !ok {
			t.Errorf("unexpected file %q", k)
		}
	}
}

func dirToMap(p string) map[string][]byte {
	m := make(map[string][]byte)
	walkFn := func(p string, d fs.FileInfo, perr error) error {
		if perr != nil {
			return perr
		}
		// XXX trim off the tempdir properly
		rel := strings.TrimLeftFunc(strings.TrimLeft(p, "/"), trimFirstDir())
		rel = strings.TrimLeftFunc(rel, trimFirstDir())
		rel = strings.TrimLeftFunc(rel, trimFirstDir())
		if d.IsDir() {
			m[rel] = nil
		} else {
			f, err := os.Open(p)
			if err != nil {
				return err
			}
			defer f.Close()
			h := sha256.New()
			if _, err := io.Copy(h, f); err != nil {
				return err
			}
			m[rel] = h.Sum(nil)
		}
		return nil
	}
	if err := filepath.Walk(p, walkFn); err != nil {
		panic(err)
	}
	return m
}
