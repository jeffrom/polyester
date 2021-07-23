package testenv

import (
	"context"
	"os"
	"os/exec"
	"testing"
)

func GitBare(t testing.TB, p string) {
	t.Helper()
	if _, err := os.Stat(p); err == nil {
		t.Fatalf("GitBare called on a file that already exists: %s", p)
	}

	if err := os.MkdirAll(p, 0755); err != nil {
		t.Fatal("GitBare MkdirAll:", err)
	}

	ctx := context.Background()
	args := []string{"init", "--bare", p}
	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	t.Logf("+ %s", cmd.Args)
	if err := cmd.Run(); err != nil {
		t.Fatal("git init failed:", err)
	}
	t.Logf("created bare git repo at %s", p)
}

func GitClone(t testing.TB, cloneURL, p string) {
	t.Helper()

	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "git", "clone", cloneURL, p)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	t.Logf("+ %s", cmd.Args)
	if err := cmd.Run(); err != nil {
		t.Fatal("git clone failed:", err)
	}
}

func GitCommitAllPush(t testing.TB, repoPath, message string) {
	t.Helper()
	if _, err := os.Stat(repoPath); err != nil {
		t.Fatalf("GitCommit called on a repo path that doesn't exist: %s", repoPath)
	}
	ctx := context.Background()

	cmd := exec.CommandContext(ctx, "git", "add", ".")
	cmd.Dir = repoPath
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	t.Logf("+ %s", cmd.Args)
	if err := cmd.Run(); err != nil {
		t.Fatal("git add failed:", err)
	}

	cmd = exec.CommandContext(ctx, "git", "-c", "commit.gpgsign=false", "commit", "--allow-empty", "-am", message)
	cmd.Dir = repoPath
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	t.Logf("+ %s", cmd.Args)
	if err := cmd.Run(); err != nil {
		t.Fatal("git commit failed:", err)
	}

	cmd = exec.CommandContext(ctx, "git", "push", "origin", "master")
	cmd.Dir = repoPath
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	t.Logf("+ %s", cmd.Args)
	if err := cmd.Run(); err != nil {
		t.Fatal("git push failed:", err)
	}
	t.Logf("created commit on git repo at %s", repoPath)
}
