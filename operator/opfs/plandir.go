package opfs

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/bmatcuk/doublestar/v4"
)

type PlanDir interface {
	fs.GlobFS
	fs.StatFS
	fs.ReadDirFS
	fs.ReadFileFS
	Join(paths ...string) string
}

type fsPlanDir struct {
	dir   string
	dirFS fs.FS
}

func NewPlanDirFS(dir string) fsPlanDir {
	return fsPlanDir{
		dir:   dir,
		dirFS: os.DirFS(dir),
	}
}

func (pd fsPlanDir) Open(name string) (fs.File, error) { return pd.dirFS.Open(name) }

func (pd fsPlanDir) Stat(name string) (fs.FileInfo, error) {
	p := filepath.Join(pd.dir, name)
	return os.Stat(p)
}

func (pd fsPlanDir) ReadDir(name string) ([]fs.DirEntry, error) {
	p := filepath.Join(pd.dir, name)
	return os.ReadDir(p)
}

func (pd fsPlanDir) ReadFile(name string) ([]byte, error) {
	p := filepath.Join(pd.dir, name)
	return os.ReadFile(p)
}

func (pd fsPlanDir) Glob(pattern string) ([]string, error) {
	return doublestar.Glob(pd, pattern)
}

func (pd fsPlanDir) Join(paths ...string) string {
	return filepath.Join(append([]string{pd.dir}, paths...)...)
}
