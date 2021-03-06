// Package shell handles parsing shell scripts in plan files.
package shell

import (
	"bytes"
	"context"
	"io"

	"mvdan.cc/sh/v3/syntax"
)

type Parser struct {
	raw []byte
	*syntax.File
}

func Parse(r io.Reader) (*Parser, error) {
	buf := bytes.NewBuffer(nil)
	if _, err := io.Copy(buf, io.LimitReader(r, 1024*1024*256)); err != nil {
		return nil, err
	}
	b := buf.Bytes()

	f, err := syntax.NewParser().Parse(bytes.NewReader(b), "plan")
	if err != nil {
		return nil, err
	}
	return &Parser{raw: b, File: f}, nil
}

func (psh *Parser) Compile(ctx context.Context) error {
	// printer := syntax.NewPrinter()
	// for i, stmt := range psh.Stmts {
	for _, stmt := range psh.Stmts {
		switch t := stmt.Cmd.(type) {
		case *syntax.CallExpr:
			if len(t.Args) == 0 {
				continue
			}
			if t.Args[0].Lit() == "set" {
				continue
			}
			// fmt.Printf("WOOP %+v\n", t.Args[0].Lit())
			// fmt.Printf("lits %+v\n", literals(t.Args))
		}

		// fmt.Printf("Cmd %d: %-20T - ", i, stmt.Cmd)
		// printer.Print(os.Stdout, stmt.Cmd)
		// fmt.Println()
	}
	return nil
}

func (psh *Parser) Extract() ([]*syntax.CallExpr, error) {
	var res []*syntax.CallExpr
	for _, stmt := range psh.Stmts {
		switch t := stmt.Cmd.(type) {
		case *syntax.CallExpr:
			if len(t.Args) == 0 {
				continue
			}
			if t.Args[0].Lit() == "set" {
				continue
			}
			// fmt.Printf("WOOP %+v\n", t.Args[0].Lit())
			// fmt.Printf("lits %+v\n", Literals(t.Args))
			if arg := t.Args[0].Lit(); arg == "P" || arg == "polyester" {
				res = append(res, t)
			}
		}
	}
	return res, nil
}

func Literals(args []*syntax.Word) []string {
	res := make([]string, len(args))
	for i, arg := range args {
		res[i] = arg.Lit()
	}
	return res
}
