package manifest

import (
	"bytes"
	"fmt"
	"io"
	"io/fs"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

type tarFS struct {
	root  string
	files []*tarFile
}

func newTarFS(root string) *tarFS {
	return &tarFS{
		root: root,
	}
}

func (tfs *tarFS) Open(name string) (fs.File, error) {
	f, found := tfs.lookup(name)
	if !found {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	if !f.IsDir() {
		f.r = bytes.NewReader(f.contents)
	}
	return f, nil
}

func (tfs *tarFS) ReadDir(name string) ([]fs.DirEntry, error) {
	f, ok := tfs.lookup(name)
	if !ok {
		return nil, &fs.PathError{Op: "open", Path: name, Err: fs.ErrNotExist}
	}
	res := make([]fs.DirEntry, len(f.files))
	for i, file := range f.files {
		res[i] = file
	}
	sort.Slice(res, func(i, j int) bool {
		return res[i].Name() < res[j].Name()
	})
	return res, nil
}

func (tfs *tarFS) AddFile(f *tarFile) {
	tfs.ensureDirs(f.name)

	if nextDir, ok := tfs.lookupDir(f.name); ok {
		if nextDir == nil {
			tfs.files = append(tfs.files, f)
		} else {
			nextDir.files = append(nextDir.files, f)
		}
	} else {
		panic("failed to find dir for: " + f.name)
	}
}

func (tfs *tarFS) ensureDirs(name string) {
	parts, _ := splitParts(name)

	curr := &tarFile{
		name:  tfs.root,
		files: tfs.files,
	}
	orig := curr
	for _, currPart := range parts {
		var next *tarFile
		found := false
		for _, f := range curr.files {
			if f.name == currPart && f.IsDir() {
				next = f
				found = true
				break
			}
		}
		if !found {
			next = &tarFile{
				name:    currPart,
				mode:    0755 | fs.ModeDir,
				modTime: time.Now(),
			}
			curr.files = append(curr.files, next)
		}

		curr = next
	}
	tfs.files = orig.files
}

func hasDir(name string, files []*tarFile) bool {
	for _, file := range files {
		if name == file.name {
			return true
		}
	}
	return false
}

func (tfs *tarFS) lookupDir(name string) (*tarFile, bool) {
	parts, _ := splitParts(name)
	var curr *tarFile
	currFiles := tfs.files
	for _, currPart := range parts {
		var f *tarFile
		found := false
		for _, cand := range currFiles {
			if cand.name == currPart {
				f = cand
				found = true
				break
			}
		}
		if !found {
			return f, false
		}

		curr = f
		currFiles = f.files
	}
	return curr, true
}

func (tfs *tarFS) lookup(name string) (*tarFile, bool) {
	dir, ok := tfs.lookupDir(name)
	if !ok {
		return nil, false
	}
	files := tfs.files
	if dir != nil {
		files = dir.files
	}
	_, toMatch := filepath.Split(name)
	for _, cand := range files {
		_, candFile := filepath.Split(cand.name)
		if candFile == toMatch {
			return cand, true
		}
	}
	return nil, false
}

func splitParts(p string) ([]string, string) {
	dir, file := filepath.Split(p)
	cleaned := filepath.Clean(dir)
	if cleaned == "." {
		return nil, file
	}
	parts := strings.Split(cleaned, string(filepath.Separator))
	return parts, file
}

type tarFile struct {
	name    string
	mode    fs.FileMode
	modTime time.Time

	contents []byte
	r        io.Reader

	files []*tarFile
}

func (tf tarFile) String() string {
	return fmt.Sprintf("%T<name: %s (mode: %s)>", tf, tf.name, tf.mode)
}

func (tf tarFile) Stat() (fs.FileInfo, error) { return tf, nil }

func (tf tarFile) Read(b []byte) (int, error) { return tf.r.Read(b) }

func (tf tarFile) Close() error {
	tf.r = nil
	tf.contents = nil
	return nil
}

func (tf tarFile) Name() string       { return filepath.Base(tf.name) }
func (tf tarFile) Size() int64        { return int64(len(tf.contents)) }
func (tf tarFile) Mode() fs.FileMode  { return tf.mode }
func (tf tarFile) IsDir() bool        { return tf.mode.IsDir() }
func (tf tarFile) ModTime() time.Time { return tf.modTime }
func (tf tarFile) Sys() interface{}   { return nil }

func (tf tarFile) Type() fs.FileMode          { return tf.mode.Type() }
func (tf tarFile) Info() (fs.FileInfo, error) { return tf, nil }
