package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/jeffrom/polyester/testenv"
)

func TestDocker(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping because -short")
	}
	dockerTest := os.Getenv("DOCKER_TEST")
	if err := testenv.DockerAvailable(); err != nil && dockerTest == "" {
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
	runDockerTest(t, "template", "Template")
}

func TestNoop(t *testing.T) {
	checkTestFilter(t, "noop")

	planPath := testenv.Path(filepath.Join("testdata", "noop"))
	for i := 0; i < 3; i++ {
		if err := run([]string{"polyester", "apply", planPath}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestBasic(t *testing.T) {
	checkTestFilter(t, "basic")

	planPath := testenv.Path(filepath.Join("testdata", "basic"))
	for i := 0; i < 3; i++ {
		if err := run([]string{"polyester", "apply", planPath}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestUseradd(t *testing.T) {
	checkTestFilter(t, "useradd")

	planPath := testenv.Path(filepath.Join("testdata", "useradd"))
	for i := 0; i < 3; i++ {
		if err := run([]string{"polyester", "apply", planPath}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestAptInstall(t *testing.T) {
	checkTestFilter(t, "apt-install")

	planPath := testenv.Path(filepath.Join("testdata", "apt-install"))
	for i := 0; i < 3; i++ {
		if err := run([]string{"polyester", "apply", planPath}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestCopy(t *testing.T) {
	checkTestFilter(t, "copy")
	planPath := testenv.Path(filepath.Join("testdata", "copy"))
	for i := 0; i < 3; i++ {
		if err := run([]string{"polyester", "apply", planPath}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestPcopy(t *testing.T) {
	checkTestFilter(t, "pcopy")
	planPath := testenv.Path(filepath.Join("testdata", "pcopy"))
	for i := 0; i < 3; i++ {
		if err := run([]string{"polyester", "apply", planPath}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestAtomicCopy(t *testing.T) {
	checkTestFilter(t, "atomic-copy")
	planPath := testenv.Path(filepath.Join("testdata", "atomic-copy"))
	for i := 0; i < 3; i++ {
		if err := run([]string{"polyester", "apply", planPath}); err != nil {
			t.Fatal(err)
		}
	}
}

func TestTemplate(t *testing.T) {
	checkTestFilter(t, "template")
	planPath := testenv.Path(filepath.Join("testdata", "template"))
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
	// projectDir := testenv.Path("")
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

func buildTestImage(t testing.TB) {
	projectDir := testenv.Path("")
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
