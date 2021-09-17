package main

import (
	"context"
	"fmt"
	"os"

	"github.com/jeffrom/polyester/cmd/polyester/commands"
	"github.com/jeffrom/polyester/stdio"
)

func main() {
	if err := run(os.Args); err != nil {
		fmt.Fprintln(os.Stderr, "Error:", err)
		os.Exit(1)
	}
}

func run(rawArgs []string) error {
	ctx := stdio.SetContext(context.Background(), &stdio.StdIO{})
	return commands.ExecArgs(ctx, rawArgs[1:])
}
