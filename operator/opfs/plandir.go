package opfs

import (
	"errors"
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
	Subplan() string
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
	// fmt.Printf("WithSubplan: %q\n", spdir)
	return fsPlanDir{
		dir:   pd.dir,
		dirFS: pd.dirFS,
		spdir: spdir,
	}
}

func (pd fsPlanDir) cleanPath(name string) string {
	name = strings.TrimPrefix(name, pd.dir+"/")
	return name
	// if filepath.IsAbs(name) {
	// 	return name
	// }
	// return filepath.Join(pd.dir, name)
}

func (pd fsPlanDir) checkPath(name string) error {
	if filepath.IsAbs(name) {
		if !strings.HasPrefix(filepath.Clean(name), filepath.Clean(pd.dir)) {
			return &fs.PathError{Op: "plandir-open", Path: name, Err: fs.ErrInvalid}
		}
	}
	return nil
}

func (pd fsPlanDir) Open(name string) (fs.File, error) {
	// if err := pd.checkPath(name); err != nil {
	// 	return nil, err
	// }
	if filepath.IsAbs(name) {
		return pd.dirFS.Open(name)
	}
	p := pd.cleanPath(name)
	fmt.Println("plandir Open:", name, p)
	return pd.dirFS.Open(p)
}

func (pd fsPlanDir) Stat(name string) (fs.FileInfo, error) {
	if err := pd.checkPath(name); err != nil {
		return nil, err
	}
	if filepath.IsAbs(name) {
		return os.Stat(name)
	}
	p := pd.cleanPath(name)
	return os.Stat(p)
}

func (pd fsPlanDir) ReadDir(name string) ([]fs.DirEntry, error) {
	if err := pd.checkPath(name); err != nil {
		return nil, err
	}
	if filepath.IsAbs(name) {
		return os.ReadDir(name)
	}
	p := pd.cleanPath(name)
	return os.ReadDir(p)
}

func (pd fsPlanDir) ReadFile(name string) ([]byte, error) {
	if err := pd.checkPath(name); err != nil {
		return nil, err
	}
	if filepath.IsAbs(name) {
		return os.ReadFile(name)
	}
	p := pd.cleanPath(name)
	return os.ReadFile(p)
}

func (pd fsPlanDir) Glob(pattern string) ([]string, error) {
	if err := pd.checkPath(pattern); err != nil {
		return nil, err
	}
	return doublestar.Glob(pd, pattern)
}

func (pd fsPlanDir) Join(paths ...string) string {
	joined := filepath.Join(paths...)
	if err := pd.checkPath(joined); err != nil {
		panic(err)
	}
	if filepath.IsAbs(joined) {
		return joined
	}

	dir := pd.dir
	// if pd.spdir != "" {
	// 	dir = pd.spdir
	// 	fmt.Println("uh oh", dir)
	// }
	return filepath.Join(dir, joined)
}

func (pd fsPlanDir) Subplan() string {
	if pd.spdir == "" {
		return ""
	}

	dir := strings.TrimPrefix(pd.spdir, filepath.Join(pd.dir, "plans")+string(filepath.Separator))
	return dir
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
		// TODO something unexpected happening here
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

		parts := strings.SplitN(pat, string(filepath.Separator), 3)
		if len(parts) == 3 && parts[0] == "plans" {
			return nil, errors.New("plandir: disallowed relative access to outside plan")
			// cands = []string{filepath.Join(planDir, "plans", parts[1], kind, parts[2])}
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
