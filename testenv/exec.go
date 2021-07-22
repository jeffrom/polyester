package testenv

import (
	"context"
	"os"
	"os/exec"
	"strconv"
)

func FakeExecPlanCompiler() func(context.Context, string, ...string) *exec.Cmd {
	return func(ctx context.Context, command string, args ...string) *exec.Cmd {
		arg := []string{"-test.run=TestFakePlanCompiler", "--", command}
		arg = append(arg, args...)
		cmd := exec.Command(os.Args[0], arg...)

		env := []string{"_TEST_WANT_HELPER_PROCESS=1"}
		cmd.Env = env
		return cmd
	}
}

// FakePlanCompiler is a fake implementation of polyester that can be used in
// unit tests by calling in a test function to allow for compilation of shell
// plans by writing a script that calls the test into $PATH.
func FakePlanCompiler() {
	if os.Getenv("_TEST_WANT_HELPER_PROCESS") != "1" {
		return
	}

	codes := os.Getenv("_TEST_EXITCODE")
	if codes != "" {
		code, err := strconv.ParseInt(codes, 10, 8)
		if err != nil {
			panic(err)
		}
		defer os.Exit(int(code))
	}

	planPath := os.Getenv("_POLY_PLAN")
	if planPath == "" {
		panic("testenv: $_POLY_PLAN was empty")
	}

}
