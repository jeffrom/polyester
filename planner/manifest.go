package planner

import (
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"hash"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/jeffrom/polyester/operator/opfs"
)

// Manifest is a top-level plan that contains its own state management.  the
// absolute path of the manifest is used as the cache key. To support moving
// manifests, if a manifest is new in the current apply run, polyester checks
// if it matches any current manifests, and if it does, the state is copied
// over.
// type Manifest struct {
// 	*Plan
// 	// dir is the absolute path of the manifest that should be unique to the
// 	// local system.
// 	dir string
// }

func (r *Planner) setupState(plan *Plan, opts ApplyOpts) (string, error) {
	// 1. figure out if this is a single script run & find the manifest file
	// (the nearest parent with polyester.sh)
	mDir, err := r.findManifestDir()
	if err != nil {
		return "", err
	}
	fmt.Println("manifest dir is:", mDir)

	// 2. if the directory doesn't already exist, create it
	key := manifestKey(mDir)
	stateDir := filepath.Join(opts.StateDir, key)
	if _, err := os.Stat(stateDir); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return "", err
		}
		if err := os.MkdirAll(stateDir, 0700); err != nil {
			return "", err
		}
	}

	// 3. update the manifest checksum in the state directory
	cs, err := manifestChecksum(plan, nil)
	if err != nil {
		return "", err
	}
	if err := os.WriteFile(filepath.Join(stateDir, "checksum"), []byte(cs), 0600); err != nil {
		return "", err
	}
	return stateDir, nil
}

func (r *Planner) pruneState(plan *Plan, stateDir string) error {
	// don't prune for single subplan runs, only for manifests.
	mDir, err := r.findManifestDir()
	if err != nil {
		return err
	}
	fmt.Println("prune", r.planFile)
	if mDir != r.planDir || r.planFile != "polyester.sh" {
		fmt.Println("skipping prune since a manifest is not being executed")
		return nil
	}

	// traverse the plans, calculate all checksums, and remove anything in the
	// state dir that doesn't match one of them.

	keys, err := r.gatherOpKeys(plan, stateDir, nil)
	if err != nil {
		return err
	}

	walkFn := func(p string, d fs.DirEntry, perr error) error {
		if perr != nil {
			return perr
		}
		if d.IsDir() {
			return nil
		}
		if d.Name() == "checksum" {
			return nil
		}
		if !keys[d.Name()] {
			fmt.Println("pruning old state:", p)
			if err := os.Remove(p); err != nil {
				return err
			}
		}
		return nil
	}

	if err := fs.WalkDir(opfs.New(stateDir), stateDir, walkFn); err != nil {
		return fmt.Errorf("failed to walk state dir: %w", err)
	}
	return nil
}

func (r *Planner) findManifestDir() (string, error) {
	mDir := r.planDir
	for mDir != "/" && mDir != "" {
		cand := filepath.Join(mDir, "polyester.sh")
		if _, err := os.Stat(cand); err == nil {
			break
		}
		mDir, _ = filepath.Split(filepath.Clean(mDir))
	}
	if mDir == "/" || mDir == "" {
		return r.planDir, errors.New("manifest: polyester.sh not found")
	}
	return mDir, nil
}

func (r *Planner) gatherOpKeys(plan *Plan, stateDir string, keys map[string]bool) (map[string]bool, error) {
	if keys == nil {
		keys = make(map[string]bool)
	}

	for _, op := range plan.Operations {
		key, err := opCacheKey(op.Info().Data())
		if err != nil {
			return nil, err
		}
		if key != "" {
			keys[key] = true
		}
	}

	for _, sp := range plan.Dependencies {
		var err error
		keys, err = r.gatherOpKeys(sp, stateDir, keys)
		if err != nil {
			return nil, err
		}
	}
	for _, sp := range plan.Plans {
		var err error
		keys, err = r.gatherOpKeys(sp, stateDir, keys)
		if err != nil {
			return nil, err
		}
	}

	return keys, nil
}

func manifestKey(dir string) string {
	return strings.Trim(
		strings.ReplaceAll(dir, string(filepath.Separator), "-"),
		"-",
	)
}

// manifestChecksum returns a checksum of the manifest suitable for determining
// equality to other manifests.
func manifestChecksum(plan *Plan, h hash.Hash) (string, error) {
	parent := false
	if h == nil {
		h = sha256.New()
		parent = true
	}
	for _, op := range plan.Operations {
		data := op.Info().Data()
		if _, err := h.Write([]byte(data.OpName)); err != nil {
			return "", err
		}
		if err := data.Encode(h); err != nil {
			return "", err
		}
	}

	for _, sp := range plan.Dependencies {
		if _, err := manifestChecksum(sp, h); err != nil {
			return "", err
		}
	}
	for _, sp := range plan.Plans {
		if _, err := manifestChecksum(sp, h); err != nil {
			return "", err
		}
	}

	if !parent {
		return "", nil
	}
	key := base64.URLEncoding.EncodeToString(h.Sum(nil))
	return nonAlnumRE.ReplaceAllLiteralString(key, ""), nil
}
