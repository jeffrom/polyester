// Package format contains code to control the look of planner output.
package format

import (
	"fmt"
	"io"
)

// type Interface struct {
// }

type DefaultFormatter struct{}

func (fm DefaultFormatter) OpComplete(w io.Writer, name string, empty, changed, dirty, executed bool) error {
	stateLabel := "unchanged"
	switch {
	case changed || dirty:
		stateLabel = "changed"
	case empty && executed && !dirty:
		stateLabel = "unchanged"
	case empty:
		stateLabel = "empty"
	}
	execLabel := ""
	if executed {
		execLabel = "X"
	}

	fmt.Fprintf(w, "%25s: [ %1s ] [ %9s ]\n", name, execLabel, stateLabel)
	return nil
}
