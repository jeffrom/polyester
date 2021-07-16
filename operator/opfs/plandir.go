package opfs

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/bmatcuk/doublestar/v4"
)

type PlanDir interface {
	fs.GlobFS
	fs.StatFS
	fs.ReadDirFS
	fs.ReadFileFS
	Join(paths ...string) string
	WithSubplan(spdir string) PlanDir
	Resolve(kind string, pats []string) ([]string, error)
}

type fsPlanDir struct {
	dir   string
	spdir string
	dirFS fs.FS
}

func NewPlanDirFS(dir string) fsPlanDir {
	return fsPlanDir{
		dir:   dir,
		dirFS: os.DirFS(dir),
	}
}

func (pd fsPlanDir) WithSubplan(spdir string) PlanDir {
	return fsPlanDir{
		dir:   pd.dir,
		dirFS: pd.dirFS,
		spdir: spdir,
	}
}

func (pd fsPlanDir) Open(name string) (fs.File, error) {
	name = strings.TrimPrefix(name, pd.dir+"/")
	p := filepath.Join(pd.dir, name)
	return pd.dirFS.Open(p)
}

func (pd fsPlanDir) Stat(name string) (fs.FileInfo, error) {
	name = strings.TrimPrefix(name, pd.dir+"/")
	p := filepath.Join(pd.dir, name)
	return os.Stat(p)
}

func (pd fsPlanDir) ReadDir(name string) ([]fs.DirEntry, error) {
	name = strings.TrimPrefix(name, pd.dir+"/")
	p := filepath.Join(pd.dir, name)
	return os.ReadDir(p)
}

func (pd fsPlanDir) ReadFile(name string) ([]byte, error) {
	name = strings.TrimPrefix(name, pd.dir+"/")
	p := filepath.Join(pd.dir, name)
	return os.ReadFile(p)
}

func (pd fsPlanDir) Glob(pattern string) ([]string, error) {
	return doublestar.Glob(pd, pattern)
}

func (pd fsPlanDir) Join(paths ...string) string {
	return filepath.Join(append([]string{pd.dir}, paths...)...)
}

// Resolve returns real path to files located in a files/ directory in the plan
// dir and matching pat (todo secret).
//
// plan paths are resolved like so (secrets should be the same): absolute paths
// (starting with /) are disallowed. if the path starts with "./", the dot will
// be expanded to the absolute path of the manifest root. 1. files/ in the
// current plan being executed 2. check parents files/ dir until the manifest
// dir is reached
func (pd fsPlanDir) Resolve(kind string, pats []string) ([]string, error) {
	planDir := pd.dir
	spDir := pd.spdir

	// fmt.Println("uhhh", pd.dir, pd.spdir, pats)
	var res []string
	for _, pat := range pats {
		if len(pat) > 0 && pat[0] == filepath.Separator {
			return nil, fmt.Errorf("plandir copy: absolute path disallowed (%s)", pat)
		}
		if len(pat) > 1 && pat[0] == '.' && pat[1] == filepath.Separator {
			pat = filepath.Join(planDir, pat)
		}

		cands := []string{
			filepath.Join(planDir, kind, pat),
		}
		if spDir != "" {
			cands = []string{
				filepath.Join(spDir, kind, pat),
				filepath.Join(planDir, kind, pat),
			}
		}

		found := false
		for _, cand := range cands {
			// fmt.Println("cand:", cand)
			matches, err := doublestar.Glob(pd, cand)
			// fmt.Println("matches:", err, len(matches), matches)
			cleaned := make([]string, len(matches))
			for i, m := range matches {
				cleaned[i] = strings.TrimPrefix(m, planDir+string(filepath.Separator))
			}
			if err == nil && len(cleaned) > 0 {
				res = append(res, cleaned...)
				found = true
				break
			}
		}

		if !found {
			return nil, fmt.Errorf("plandir copy: %s not found", pat)
		}
	}

	return res, nil
}
