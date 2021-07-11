// Package shell handles parsing shell scripts in plan files.
package shell

import (
	"context"
	"fmt"
	"io"
	"os"

	"mvdan.cc/sh/v3/syntax"
)

type Parser struct {
	*syntax.File
}

func Parse(r io.Reader) (*Parser, error) {
	f, err := syntax.NewParser().Parse(r, "plan")
	if err != nil {
		return nil, err
	}
	return &Parser{File: f}, nil
}

func (psh *Parser) Compile(ctx context.Context) error {
	printer := syntax.NewPrinter()
	for i, stmt := range psh.Stmts {
		fmt.Printf("Cmd %d: %-20T - ", i, stmt.Cmd)
		printer.Print(os.Stdout, stmt.Cmd)
		fmt.Println()
	}
	return nil
}
