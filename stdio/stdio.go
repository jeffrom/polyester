// Package stdio manages standard io in a way that's easily mockable in tests
// while also not depending on overriding os.Stdin, os.Stdout, and os.Stderr.
package stdio

import (
	"context"
	"io"
	"os"
)

type contextKey string

var stdioKey = contextKey("stdio")

type StdIO struct {
	In      io.Reader
	Out     io.Writer
	Err     io.Writer
	Quiet   bool
	Verbose bool
	scopes  []string
}

func (o StdIO) Stdin() io.Reader {
	if o.In != nil {
		return o.In
	}
	return os.Stdin
}

func (o StdIO) Stdout() io.Writer {
	if o.Out != nil {
		return o.Out
	}
	return os.Stdout
}

func (o StdIO) Stderr() io.Writer {
	if o.Err != nil {
		return o.Err
	}
	return os.Stderr
}

func (o StdIO) WithScope(scopes ...string) StdIO {
	o.scopes = scopes
	return o
}

func (o StdIO) AppendScope(scopes ...string) StdIO {
	o.scopes = append(o.scopes, scopes...)
	return o
}

func (o StdIO) ClearScope() StdIO {
	o.scopes = nil
	return o
}

func SetContext(ctx context.Context, o *StdIO) context.Context {
	return context.WithValue(ctx, stdioKey, o)
}

func FromContext(ctx context.Context) *StdIO {
	if ctx == nil {
		panic("stdio: context was nil")
	}
	iv := ctx.Value(stdioKey)
	if iv == nil {
		panic("stdio: context stdio value missing")
	}
	return iv.(*StdIO)
}

func Stdin(ctx context.Context) io.Reader  { return ctx.Value(stdioKey).(StdIO).Stdin() }
func Stdout(ctx context.Context) io.Writer { return ctx.Value(stdioKey).(StdIO).Stdout() }
func Stderr(ctx context.Context) io.Writer { return ctx.Value(stdioKey).(StdIO).Stderr() }
