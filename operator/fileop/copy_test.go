package fileop

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/jeffrom/polyester/operator"
	"github.com/jeffrom/polyester/operator/opfs"
	"github.com/jeffrom/polyester/testenv"
)

func TestCopy(t *testing.T) {
	tcs := []struct {
		name          string
		opts          *CopyOpts
		expectFiles   []string
		expectNoFiles []string
		expectDirs    []string
		expectFail    bool
	}{
		{
			name:        "basic",
			opts:        destSrc("/dest/afile", "/adir/afile"),
			expectFiles: strs("/dest/afile"),
		},
		{
			name:        "no-dest-file",
			opts:        destSrc("/dest", "/adir/afile"),
			expectFiles: strs("/dest/afile"),
		},
		{
			name:        "no-dest-file-slash",
			opts:        destSrc("/dest/", "/adir/afile"),
			expectFiles: strs("/dest/afile"),
			expectDirs:  strs("/dest"),
		},
		{
			name:        "multi",
			opts:        destSrc("/dest", "/adir/afile", "/adir/bfile"),
			expectFiles: strs("/dest/afile"),
		},
		{
			name:       "missing-srcdir",
			opts:       destSrc("/dest/afile", "/not-existing-dir/afile"),
			expectFail: true,
		},
		{
			name:          "multi-missing-src",
			opts:          destSrc("/dest", "/not-existing-dir/afile", "/not-existing-dir/bfile"),
			expectNoFiles: strs("/dest/afile"),
			expectFail:    true,
		},
		{
			name:          "multi-missing-src-2",
			opts:          destSrc("/dest", "/adir/afile", "/not-existing-dir/bfile"),
			expectNoFiles: strs("/dest/afile"),
			expectFail:    true,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			tmpdir := testenv.TempDir(t, "")
			defer testenv.RemoveOnSuccess(t, tmpdir)
			testenv.Mkdirs(t, 0755, filepath.Join(tmpdir, "adir"), filepath.Join(tmpdir, "dest"))
			testenv.WriteFile(t, filepath.Join(tmpdir, "adir", "afile"), "a")
			testenv.WriteFile(t, filepath.Join(tmpdir, "adir", "bfile"), "b")

			opts := &CopyOpts{}
			if tc.opts != nil {
				opts = tc.opts
			}
			op := &Copy{Args: opts}

			ctx := context.Background()
			ofs := opfs.New(tmpdir)
			octx := operator.NewContext(ctx, ofs, nil, nil)
			err := op.Run(octx)
			if !tc.expectFail && err != nil {
				t.Fatal("copy unexpectedly failed:", err)
			} else if tc.expectFail && err == nil {
				t.Fatal("copy unexpectedly succeeded")
			}

			for _, expectFile := range tc.expectFiles {
				fullPath := ofs.Join(expectFile)
				if info, err := os.Stat(fullPath); err != nil {
					t.Errorf("expected file: %v", err)
				} else if info.IsDir() {
					t.Errorf("expected file but got a directory: %s", expectFile)
				}
			}

			for _, expectDir := range tc.expectDirs {
				if info, err := os.Stat(ofs.Join(expectDir)); err != nil {
					t.Errorf("expected file: %v", err)
				} else if !info.IsDir() {
					t.Errorf("expected directory but got a file: %s", expectDir)
				}
			}

			for _, expectNoFile := range tc.expectNoFiles {
				if _, err := os.Stat(ofs.Join(expectNoFile)); err == nil {
					t.Errorf("expected file %s not to exist", expectNoFile)
				}
			}
		})
	}
}

func destSrc(dest string, sources ...string) *CopyOpts {
	return &CopyOpts{Dest: dest, Sources: sources}
}

func strs(s ...string) []string { return s }
