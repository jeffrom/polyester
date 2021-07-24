// Package gitop contains operators that use git.
package gitop

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
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
		// if in detached mode, this will just be a bare commit id. use git
		// remote show to get the ref.
		ref, err = getRemoteDefaultBranch(octx, repoDir)
		if err != nil {
			return "", "", fmt.Errorf("failed to get ref (tried .git/HEAD too): %w", err)
		}

		if ref == "" {
			return "", "", fmt.Errorf("failed to get ref from .git/HEAD: %s", string(b))
		}
	}

	// we're not in detached mode, so we got the ref (ie .git/HEAD was
	// "ref: refs/heads/master")
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
	fmt.Println("+", cmd.Args)
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

var remoteHeadRE = regexp.MustCompile(`HEAD branch: (.*)`)

func getRemoteDefaultBranch(octx operator.Context, repoDir string) (string, error) {
	cmd := exec.CommandContext(octx.Context, "git", "remote", "show", "origin")
	fmt.Println("+", cmd.Args)
	cmd.Dir = repoDir
	outb := &bytes.Buffer{}
	cmd.Stderr = os.Stderr
	cmd.Stdout = outb
	if err := cmd.Run(); err != nil {
		return "", err
	}

	b := outb.Bytes()
	m := remoteHeadRE.FindSubmatch(b)
	if m == nil {
		return "", fmt.Errorf("failed to get remote ref: %q", string(b))
	}
	return fmt.Sprintf("refs/heads/%s", string(bytes.TrimSpace(m[1]))), nil
}
