package planner

import (
	"fmt"
	"io"
)

// TODO operation should output smth like this (or maybe just tabwriter):
// useradd:			[   ] [ unchanged ] [          ]
// sh:				[ X ] [ unchanged ] [  success ]
// atomic-copy:		[ X ] [ changed   ] [  success ]
// apt-install:		[ X ] [ empty     ] [   failed ] error: exit status 1 (apt install abc)
// touch:			[   ] [ unchanged ] [          ]
//
// if it fails, print any output (logs, combined output from shell) from the
// operation (at the end of the run, or right when it happens?)

func formatOpComplete(w io.Writer, name string, empty, changed, dirty, executed bool) error {
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
