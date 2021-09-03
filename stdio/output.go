package stdio

import (
	"fmt"
	"strings"
)

func (o StdIO) Printf(msg string, args ...interface{}) {
	fmt.Fprintf(o.Stdout(), msg, args...)
}

func (o StdIO) Println(args ...interface{}) {
	fmt.Fprintln(o.Stdout(), args...)
}

func (o StdIO) Info(args ...interface{}) {
	if o.Quiet {
		return
	}
	fmt.Fprintln(o.Stdout(), args...)
}

func (o StdIO) Infof(msg string, args ...interface{}) {
	if o.Quiet {
		return
	}
	fmt.Fprintf(o.Stdout(), msg+"\n", args...)
}

func (o StdIO) Debug(args ...interface{}) {
	if !o.Verbose {
		return
	}
	msg := fmt.Sprintf("%sWARNING:", fmtScopes(o.scopes))
	fmt.Fprintln(o.Stdout(), append([]interface{}{msg}, args...)...)
}

func (o StdIO) Debugf(msg string, args ...interface{}) {
	if !o.Verbose {
		return
	}
	msg = fmt.Sprintf("%s%s", fmtScopes(o.scopes), msg)
	fmt.Fprintf(o.Stdout(), msg+"\n", args...)
}

func (o StdIO) Warning(args ...interface{}) {
	fmt.Fprintln(o.Stderr(), append([]interface{}{"WARNING:"}, args...)...)
}

func (o StdIO) Warningf(msg string, args ...interface{}) {
	fmt.Fprintf(o.Stderr(), "WARNING: "+msg+"\n", args...)
}

func fmtScopes(scopes []string) string { return strings.Join(scopes, ":") }
