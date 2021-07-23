// Package gitop contains operators that use git.
package gitop

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/jeffrom/polyester/operator"
)

func getCurrentCommit(octx operator.Context, repoDir string) (string, string, error) {
	headPath := filepath.Join(repoDir, ".git", "HEAD")
	headInfo, err := octx.FS.Stat(headPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", "", err
	}
	if headInfo == nil {
		return "", "", nil
	}
	b, err := octx.FS.ReadFile(headPath)
	if err != nil {
		return "", "", err
	}
	ref := getHeadRef(b)
	if ref == "" {
		return "", "", fmt.Errorf("failed to get ref from .git/HEAD: %s", string(b))
	}
	currRefPath := filepath.Join(repoDir, ".git", ref)
	_, err = octx.FS.Stat(currRefPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", "", err
	}
	currRef, err := octx.FS.ReadFile(currRefPath)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return "", "", err
	}

	currRefStr := string(bytes.TrimSpace(currRef))
	return currRefStr, ref, nil
}

func getLatestCommit(octx operator.Context, repoDir, ref string) (string, error) {
	_, refName := filepath.Split(ref)
	cmd := exec.CommandContext(octx.Context, "git", "rev-parse", "origin/"+refName)
	cmd.Dir = repoDir
	outb := &bytes.Buffer{}
	cmd.Stderr = os.Stderr
	cmd.Stdout = outb
	if err := cmd.Run(); err != nil {
		return "", err
	}

	remoteHead := strings.TrimSpace(outb.String())
	return remoteHead, nil
}
