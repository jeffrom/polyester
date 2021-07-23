package format

import (
	"io"
	"strings"
	"text/tabwriter"
)

func NewTabWriter(w io.Writer) *tabwriter.Writer {
	var flags uint // | tabwriter.Debug
	padding := 3
	return tabwriter.NewWriter(w, 4, 4, padding, ' ', flags)
}

func WriteTabHeader(w io.Writer, cols ...string) {
	for i, col := range cols {
		if i > 0 {
			io.WriteString(w, "\t")
		}
		io.WriteString(w, strings.ToUpper(col))
	}
	io.WriteString(w, "\n")
}

func WriteTabRow(w io.Writer, cols ...string) {
	for i, col := range cols {
		if i > 0 {
			io.WriteString(w, "\t")
		}
		io.WriteString(w, col)
	}
	io.WriteString(w, "\t\n")
}

func Bool(v bool) string {
	if v {
		return "[ X ]"
	}
	return "[  ]"
}
