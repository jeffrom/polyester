package shell

import (
	"bytes"

	"github.com/spf13/cobra"
	"mvdan.cc/sh/v3/syntax"

	"github.com/jeffrom/polyester/operator/shellop"
)

func (psh *Parser) ConvertShellOp() ([]byte, error) {
	buf := bytes.NewBuffer(nil)
	write := func(b []byte) {
		// fmt.Printf("write: %q\n", string(b))
		buf.WriteString(string(b))
	}

	var magicMode bool
	var end uint
	for i, stmt := range psh.Stmts {
		off := stmt.Position.Offset()
		end = stmt.End().Offset() + 1
		if i == 0 {
			write(psh.raw[:off])
		}
		// fmt.Printf("chonk: %q\n", psh.raw[off:end])

		callExpr, ok := stmt.Cmd.(*syntax.CallExpr)
		if !ok {
			write(psh.raw[off:end])
			continue
		}

		isSH := isSHCall(callExpr)
		isMagic := false
		if isSH {
			var err error
			isMagic, err = psh.isMagicSH(callExpr)
			if err != nil {
				return nil, err
			}
		}
		magicDone := magicMode && isPolyesterCall(callExpr)

		if magicDone {
			magicMode = false
			write([]byte("\"; "))
		}

		if isMagic {
			next := copySlice(psh.raw[off:end])
			next = bytes.TrimRight(next, " \n")
			write(next)
			write([]byte(" \""))
			magicMode = true
		} else {
			write(psh.raw[off:end])
		}
	}
	// last bit, finish the magic string if applicable
	if buf := psh.raw[end:]; len(buf) > 0 {
		write(buf)
	}
	if magicMode {
		write([]byte("\"\n"))
	}

	// b := bytes.Join(res, nil)
	b := buf.Bytes()
	// fmt.Println()
	// fmt.Println()
	// fmt.Printf("result: ---\n%s\n---\n", string(b))
	return b, nil
}

func (psh *Parser) isMagicSH(callExpr *syntax.CallExpr) (bool, error) {
	rawArgs := make([]string, len(callExpr.Args))
	for i, arg := range callExpr.Args {
		rawArgs[i] = string(psh.raw[arg.Pos().Offset():arg.End().Offset()])
	}
	// fmt.Printf("raw args: %q\n", rawArgs)
	cmdArgs := rawArgs[2:]
	shop := &shellop.Shell{Args: &shellop.ShellOpts{}, NoValidateArgs: true}
	shopData := shop.Info().Data()
	shopCmd := shopData.Command.Command
	found := false
	shopCmd.RunE = func(cmd *cobra.Command, args []string) error {
		found = len(args) == 0
		return nil
	}
	// this should be fine as long as all the flags are string
	shopCmd.SetArgs(cmdArgs)

	if fn := shopData.Command.ApplyArgs; fn != nil {
		if err := fn(shopData.Command.Command, cmdArgs, shop.Args); err != nil {
			return false, err
		}
	}
	if err := shopCmd.Execute(); err != nil {
		return false, err
	}
	return found, nil
}

func copySlice(sl []byte) []byte {
	next := make([]byte, len(sl))
	copy(next, sl)
	return next
}

func isSHCall(callExpr *syntax.CallExpr) bool {
	return isPolyesterCall(callExpr) &&
		len(callExpr.Args) >= 2 &&
		callExpr.Args[1].Lit() == "sh"
}

func isPolyesterCall(callExpr *syntax.CallExpr) bool {
	return len(callExpr.Args) > 0 && (callExpr.Args[0].Lit() == "polyester" || callExpr.Args[0].Lit() == "P")
}
