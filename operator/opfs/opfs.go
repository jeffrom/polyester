// Package opfs is an fs implementation that supports stat, reads, and
// globbing.
package opfs

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"

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

func (fs FS) Open(name string) (fs.File, error) {
	// fmt.Println("opfs.FS.Open:", name)
	return fs.dirFS.Open(name)
}

func (fs FS) cleanPath(name string) string {
	if strings.HasPrefix(name, fs.root) {
		return name
	}
	// fmt.Println("before", name)
	p := strings.TrimPrefix(name, fs.root)
	if name != fs.root && !strings.HasPrefix(name, fs.root) {
		p = filepath.Join(fs.root, name)
	}
	// fmt.Println("cleaned", p)
	return p
}

func (fs FS) Stat(name string) (fs.FileInfo, error) {
	// fmt.Println("Stat", name)
	return os.Stat(fs.cleanPath(name))
}

func (fs FS) ReadDir(name string) ([]fs.DirEntry, error) {
	return os.ReadDir(fs.cleanPath(name))
}

func (fs FS) ReadFile(name string) ([]byte, error) {
	return os.ReadFile(fs.cleanPath(name))
}

func (fs FS) Glob(pattern string) ([]string, error) {
	return doublestar.Glob(fs, pattern)
}

func (fs FS) Abs(name string) string {
	name = strings.TrimPrefix(name, fs.root+"/")
	return filepath.Clean(filepath.Join(fs.root, name))
}

func (fs FS) Join(paths ...string) string {
	return filepath.Join(append([]string{fs.root}, paths...)...)
}
