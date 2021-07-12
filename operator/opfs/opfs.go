// Package opfs is an fs implementation that supports stat, reads, and
// globbing.
package opfs

import (
	"io/fs"
	"os"
	"path/filepath"

	"github.com/bmatcuk/doublestar/v4"
)

type FS struct {
	dirFS fs.FS
	root  string
}

func New(root string) FS {
	return FS{
		dirFS: os.DirFS(root),
		root:  root,
	}
}

func (fs FS) Open(name string) (fs.File, error) { return fs.dirFS.Open(name) }

func (fs FS) Stat(name string) (fs.FileInfo, error) {
	p := filepath.Join(fs.root, name)
	return os.Stat(p)
}

func (fs FS) ReadDir(name string) ([]fs.DirEntry, error) {
	p := filepath.Join(fs.root, name)
	return os.ReadDir(p)
}

func (fs FS) ReadFile(name string) ([]byte, error) {
	p := filepath.Join(fs.root, name)
	return os.ReadFile(p)
}

func (fs FS) Glob(pattern string) ([]string, error) {
	return doublestar.Glob(fs, pattern)
}

func (fs FS) Abs(name string) string {
	return filepath.Clean(filepath.Join(fs.root, name))
}

func (fs FS) Join(paths ...string) string {
	return filepath.Join(append([]string{fs.root}, paths...)...)
}
