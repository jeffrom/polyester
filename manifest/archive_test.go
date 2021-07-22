package manifest

import (
	"path/filepath"
	"testing"

	"github.com/jeffrom/polyester/testenv"
)

func TestLoadSaveArchive(t *testing.T) {
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

			abs, err := Save(man, filepath.Join(tmpdir, "archive"))
			if err != nil {
				t.Fatal("Save failed:", err)
			}

			loaded, err := LoadFile(abs)
			if err != nil {
				t.Fatal("Load failed:", err)
			}

			destdir := filepath.Join(tmpdir, "dest")
			if err := SaveDir(destdir, loaded); err != nil {
				t.Fatal("SaveDir failed:", err)
			}
			checkDirsEqual(t, mandir, destdir)
		})
	}
}
