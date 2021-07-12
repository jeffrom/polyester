// Package fileop contains filesystem-related operators.
package fileop

import (
	"crypto/sha256"
	"io"
	"os"
)

func Checksum(p string) ([]byte, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	sha := sha256.New()
	if _, err := io.Copy(sha, f); err != nil {
		return nil, err
	}

	return sha.Sum(nil), nil
}
