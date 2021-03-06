package operator

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"regexp"

	"github.com/jeffrom/polyester/state"
)

func ReadState(data *InfoData, stateDir string) (state.State, error) {
	st := state.State{}
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

func SaveState(data *InfoData, st state.State, stateDir string) error {
	key, err := opCacheKey(data)
	if err != nil {
		return err
	}
	p := filepath.Join(stateDir, key)
	return st.WriteFile(p)
}

var nonAlnumRE = regexp.MustCompile(`[^A-Za-z0-9]`)

func opCacheKey(data *InfoData) (string, error) {
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
