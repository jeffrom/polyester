// Package executil wraps some functions in the exec package to ease testing
// and common subprocess use cases.
package executil

import (
	"context"
	"os/exec"
)

// CommandContext is initialized to exec.CommandContext. It is intended to be
// overridden in tests.
var CommandContext = exec.CommandContext

func SetCommand(fn func(context.Context, string, ...string) *exec.Cmd) {
	CommandContext = fn
}

func ResetCommand() { CommandContext = exec.CommandContext }
