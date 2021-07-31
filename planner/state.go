package planner

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"regexp"

	"github.com/jeffrom/polyester/operator"
)

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
