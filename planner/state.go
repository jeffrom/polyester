package planner

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"

	"github.com/jeffrom/polyester/operator"
)

func pruneState(plan *Plan, stateDir string) error {
	keys := make(map[string]bool)
	for _, op := range plan.Operations {
		key, err := opCacheKey(op.Info().Data())
		if err != nil {
			return err
		}
		if key != "" {
			keys[key] = true
		}
	}

	walkFn := func(p string, d fs.DirEntry, perr error) error {
		if perr != nil {
			return perr
		}
		if d.IsDir() {
			return fmt.Errorf("unexpectedly found a directory %q", p)
		}
		fmt.Println(p)
		return nil
	}

	if err := fs.WalkDir(os.DirFS(stateDir), stateDir, walkFn); err != nil {
		return err
	}
	return nil
}

func readPrevState(data *operator.InfoData, stateDir string) (operator.State, error) {
	st := operator.State{}
	key, err := opCacheKey(data)
	if err != nil {
		return st, err
	}
	p := filepath.Join(stateDir, key)
	f, err := os.Open(p)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return st, nil
		}
		return st, err
	}
	defer f.Close()

	if err := json.NewDecoder(f).Decode(&st); err != nil {
		return st, err
	}
	return st, nil
}

func saveState(data *operator.InfoData, st operator.State, stateDir string) error {
	key, err := opCacheKey(data)
	if err != nil {
		return err
	}
	p := filepath.Join(stateDir, key)
	return st.WriteFile(p)
}

var nonAlnumRE = regexp.MustCompile(`[^A-Za-z0-9]`)

func opCacheKey(data *operator.InfoData) (string, error) {
	targb, err := json.Marshal(data.Command.Target)
	if err != nil {
		return "", err
	}
	sha := sha256.New()
	if _, err := sha.Write([]byte(data.OpName)); err != nil {
		return "", err
	}
	if _, err := sha.Write(targb); err != nil {
		return "", err
	}
	key := base64.URLEncoding.EncodeToString(sha.Sum(nil))
	return nonAlnumRE.ReplaceAllLiteralString(key, ""), nil
}
