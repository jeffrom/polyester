// Package manifest contains functions to save and load manifests.
//
// Manifests are pre-"compiled" collections of scripts, templates, and files
// that can be read and written to and from disk, and can also be compiled.
package manifest

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var allDirs = []string{
	"files",
	"templates",
	"vars",
	"secrets",
	"plans",
}

type Manifest struct {
	Metadata   *Metadata            `json:"metadata,omitempty"`
	Main       string               `json:"main,omitempty"`
	MainScript []byte               `json:"main_script,omitempty"`
	Plans      map[string]*Manifest `json:"plans,omitempty"`
	Files      map[string][]byte    `json:"files,omitempty"`
	Templates  map[string][]byte    `json:"templates,omitempty"`
	Vars       map[string][]byte    `json:"vars,omitempty"`
	Secrets    map[string][]byte    `json:"secrets,omitempty"`
}

type Metadata struct {
	Name    string `json:"name,omitempty"`
	Version string `json:"version,omitempty"`
}

func LoadDir(dir string) (*Manifest, error) {
	if info, err := os.Stat(dir); err != nil {
		return nil, err
	} else if !info.IsDir() {
		return nil, fmt.Errorf("manifest: expected path to be dir: %s", dir)
	}
	return LoadFS(os.DirFS(dir))
}

var errDone = errors.New("DONE")

func SaveDir(dir string, m *Manifest) error {
	info, err := os.Stat(dir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return err
	}
	if info != nil {
		if !info.IsDir() {
			return fmt.Errorf("manifest: path already exists and is not a directory: %s", dir)
		}

		found := false
		walkFn := func(p string, d fs.FileInfo, perr error) error {
			if perr != nil {
				return perr
			}
			found = true
			return errDone
		}
		if err := filepath.Walk(dir, walkFn); err != nil && err != errDone {
			return err
		}
		if found {
			return fmt.Errorf("manifest: path already exists and is not empty: %s", dir)
		}
	} else if err := os.Mkdir(dir, 0755); err != nil {
		return err
	}

	if err := os.WriteFile(filepath.Join(dir, m.Main), m.MainScript, 0644); err != nil {
		return err
	}

	if len(m.Files) > 0 {
		filesDir := filepath.Join(dir, "files")
		if err := writeFiles(filesDir, m.Files); err != nil {
			return err
		}
	}
	if len(m.Templates) > 0 {
		templatesDir := filepath.Join(dir, "templates")
		if err := writeFiles(templatesDir, m.Templates); err != nil {
			return err
		}
	}
	if len(m.Vars) > 0 {
		varsDir := filepath.Join(dir, "vars")
		if err := writeFiles(varsDir, m.Vars); err != nil {
			return err
		}
	}
	if len(m.Secrets) > 0 {
		secretsDir := filepath.Join(dir, "secrets")
		if err := writeFiles(secretsDir, m.Secrets); err != nil {
			return err
		}
	}

	if len(m.Plans) > 0 {
		if err := os.Mkdir(filepath.Join(dir, "plans"), 0755); err != nil {
			return err
		}
		for name, plan := range m.Plans {
			if err := SaveDir(filepath.Join(dir, "plans", name), plan); err != nil {
				return err
			}
		}
	}
	return nil
}

func LoadFS(mfs fs.FS) (*Manifest, error) {
	return loadFS(mfs, "polyester.sh")
}

func loadFS(mfs fs.FS, mainPath string) (*Manifest, error) {
	if _, err := fs.Stat(mfs, mainPath); err != nil {
		return nil, err
	}
	b, err := fs.ReadFile(mfs, mainPath)
	if err != nil {
		return nil, err
	}

	for _, kind := range allDirs {
		if info, err := fs.Stat(mfs, kind); err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}
			return nil, err
		} else if !info.IsDir() {
			return nil, fmt.Errorf("manifest: %q should be a directory", kind)
		}
	}

	plans, err := gatherPlansFS(mfs)
	if err != nil {
		return nil, fmt.Errorf("manifest: gather plans failed: %w", err)
	}
	m := &Manifest{
		Main:       mainPath,
		MainScript: b,
		Plans:      plans,
	}
	if err := gatherFilesFS(mfs, m); err != nil {
		return nil, err
	}
	return m, nil
}

func gatherPlansFS(mfs fs.FS) (map[string]*Manifest, error) {
	if _, err := fs.Stat(mfs, "plans"); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			return nil, nil
		}
		return nil, err
	}
	m := make(map[string]*Manifest)
	walkFn := func(p string, d fs.DirEntry, perr error) error {
		if perr != nil {
			return perr
		}
		if !d.IsDir() {
			return nil
		}
		planFile := filepath.Join(p, "plan.sh")
		if info, err := fs.Stat(mfs, planFile); err == nil {
			if info.IsDir() {
				return fmt.Errorf("manifest: %q should be a directory", planFile)
			}
			subfs, err := fs.Sub(mfs, p)
			if err != nil {
				return err
			}
			manifest, err := loadFS(subfs, "plan.sh")
			if err != nil {
				return err
			}
			m[strings.TrimLeftFunc(p, trimFirstDir())] = manifest
		}
		return nil
	}
	if err := fs.WalkDir(mfs, "plans", walkFn); err != nil {
		return nil, err
	}
	return m, nil
}

func gatherFilesFS(mfs fs.FS, m *Manifest) error {
	if _, err := fs.Stat(mfs, "files"); err == nil {
		fm := make(map[string][]byte)
		if err := fs.WalkDir(mfs, "files", fsGatherer(mfs, nil, fm)); err != nil {
			return err
		}
		m.Files = fm
	}

	if _, err := fs.Stat(mfs, "vars"); err == nil {
		vm := make(map[string][]byte)
		if err := fs.WalkDir(mfs, "vars", fsGatherer(mfs, nil, vm)); err != nil {
			return err
		}
		m.Vars = vm
	}

	if _, err := fs.Stat(mfs, "templates"); err == nil {
		tm := make(map[string][]byte)
		if err := fs.WalkDir(mfs, "templates", fsGatherer(mfs, []string{"yaml", "json"}, tm)); err != nil {
			return err
		}
		m.Templates = tm
	}

	if _, err := fs.Stat(mfs, "secrets"); err == nil {
		sm := make(map[string][]byte)
		if err := fs.WalkDir(mfs, "secrets", fsGatherer(mfs, []string{"age"}, sm)); err != nil {
			return err
		}
		m.Secrets = sm
	}
	return nil
}

func fsGatherer(mfs fs.FS, suffixes []string, m map[string][]byte) fs.WalkDirFunc {
	return func(p string, d fs.DirEntry, perr error) error {
		if perr != nil {
			return perr
		}
		if d.IsDir() {
			return nil
		}
		b, err := fs.ReadFile(mfs, p)
		if err != nil {
			return err
		}
		m[strings.TrimLeftFunc(p, trimFirstDir())] = b
		return nil
	}
}

func trimFirstDir() func(ch rune) bool {
	foundSlash := false
	return func(ch rune) bool {
		if ch == '/' {
			foundSlash = true
			return true
		}
		if foundSlash {
			return false
		}
		return true
	}
}

func writeFiles(dir string, files map[string][]byte) error {
	for name, b := range files {
		fdir, _ := filepath.Split(name)
		if err := os.MkdirAll(filepath.Join(dir, fdir), 0755); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(dir, name), b, 0644); err != nil {
			return err
		}
	}
	return nil
}
