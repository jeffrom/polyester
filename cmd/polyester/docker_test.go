package main

import (
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
)

func TestDocker(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping because -short")
	}
	dockerTest := os.Getenv("DOCKER_TEST")
	if err := hasDocker(); err != nil && dockerTest == "" {
		t.Skip("skipping because docker isn't available:", err)
	}

	if dockerTest == "" {
		buildTestImage(t)
	}

	runDockerTest(t, "noop", "Noop")
	runDockerTest(t, "basic", "Basic")
	runDockerTest(t, "useradd", "Useradd")
	runDockerTest(t, "apt-install", "AptInstall")
	runDockerTest(t, "copy", "Copy")
	runDockerTest(t, "pcopy", "Pcopy")
	runDockerTest(t, "atomic-copy", "AtomicCopy")
}

func TestNoop(t *testing.T) {
	checkTestFilter(t, "noop")

	planPath := Path(filepath.Join("testdata", "noop"))
	for i := 0; i < 3; i++ {
		if err := run([]string{"polyester", "apply", planPath}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestBasic(t *testing.T) {
	checkTestFilter(t, "basic")

	planPath := Path(filepath.Join("testdata", "basic"))
	for i := 0; i < 3; i++ {
		if err := run([]string{"polyester", "apply", planPath}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestUseradd(t *testing.T) {
	checkTestFilter(t, "useradd")

	planPath := Path(filepath.Join("testdata", "useradd"))
	for i := 0; i < 3; i++ {
		if err := run([]string{"polyester", "apply", planPath}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestAptInstall(t *testing.T) {
	checkTestFilter(t, "apt-install")

	planPath := Path(filepath.Join("testdata", "apt-install"))
	for i := 0; i < 3; i++ {
		if err := run([]string{"polyester", "apply", planPath}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCopy(t *testing.T) {
	checkTestFilter(t, "copy")
	planPath := Path(filepath.Join("testdata", "copy"))
	for i := 0; i < 3; i++ {
		if err := run([]string{"polyester", "apply", planPath}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPcopy(t *testing.T) {
	checkTestFilter(t, "pcopy")
	planPath := Path(filepath.Join("testdata", "pcopy"))
	for i := 0; i < 3; i++ {
		if err := run([]string{"polyester", "apply", planPath}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestAtomicCopy(t *testing.T) {
	checkTestFilter(t, "atomic-copy")
	planPath := Path(filepath.Join("testdata", "atomic-copy"))
	for i := 0; i < 3; i++ {
		if err := run([]string{"polyester", "apply", planPath}); err != nil {
			t.Fatal(err)
		}
	}
}

func checkTestFilter(t testing.TB, name string) {
	t.Helper()
	if env := os.Getenv("DOCKER_TEST"); env != name {
		t.Skip("skipping because not selected by $DOCKER_TEST filter:", env)
	}
}

func runDockerTest(t *testing.T, name, goTestName string) {
	// projectDir := Path("")
	t.Run(name, func(t *testing.T) {
		args := []string{
			"run", "--rm", "-e", "DOCKER_TEST=" + name, "jeffmartin1117/polyester:test",
			"go", "test", "-v", "-count", "1", "-run", "^Test" + goTestName + "$", "./cmd/polyester",
		}
		cmd := exec.Command("docker", args...)
		fmt.Printf("+ %v\n", cmd.Args)
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		cmd.Env = []string{
			"DOCKER_TEST=" + name,
			"PATH=/bin:/usr/bin:/usr/local/bin",
		}

		if err := cmd.Run(); err != nil {
			t.Fatal(err)
		}
	})
}

func hasDocker() error {
	if _, err := exec.LookPath("docker"); err != nil {
		return err
	}
	return nil
}

func buildTestImage(t testing.TB) {
	projectDir := Path("")
	dockerfilePath := filepath.Join(projectDir, "testdata", "test.Dockerfile")
	args := []string{
		"build", projectDir,
		"-f", dockerfilePath,
		"-t", "jeffmartin1117/polyester:test",
	}
	ctx := context.Background()
	cmd := exec.CommandContext(ctx, "docker", args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		t.Fatal(err)
	}
}

func Path(p string) string {
	dir, err := findGoMod()
	die(err)

	finalPath := filepath.Join(dir, p)
	return finalPath
}

var gomodPath string

func findGoMod() (string, error) {
	if gomodPath != "" {
		return gomodPath, nil
	}

	_, file, _, ok := runtime.Caller(1)
	if !ok {
		return "", errors.New("failed to get path of caller's file")
	}
	dir, _ := filepath.Split(file)

	for d := dir; d != "/"; d, _ = filepath.Split(filepath.Clean(d)) {
		gomodPath := filepath.Join(d, "go.mod")
		if _, err := os.Stat(gomodPath); err != nil {
			continue
		}
		gomodPath = d
		return d, nil
	}
	return "", errors.New("failed to find project root")
}

func die(err error) {
	if err != nil {
		panic(err)
	}
}
