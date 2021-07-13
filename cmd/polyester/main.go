package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jeffrom/polyester/cmd/polyester/commands"
)

func main() {
	if err := run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func run(rawArgs []string) error {
	return commands.ExecArgs(context.Background(), rawArgs[1:])
}
