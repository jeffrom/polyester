package operator

import (
	"context"
	"io/fs"
)

type ctxKey string

var gotStateKey = ctxKey("gotState")

type Context struct {
	context.Context

	Opts interface{}
	FS   FS
}

func NewContext(ctx context.Context, fs FS) Context {
	return Context{
		Context: ctx,
		FS:      fs,
	}
}

func (c Context) WithGotState(gotState bool) Context {
	ctx := context.WithValue(c.Context, gotStateKey, gotState)
	return Context{
		Context: ctx,
		Opts:    c.Opts,
		FS:      c.FS,
	}
}

func (c Context) GotState() bool {
	return c.Context.Value(gotStateKey).(bool)
}

type FS interface {
	fs.StatFS
	fs.GlobFS
	fs.ReadDirFS
	fs.ReadFileFS
	Abs(name string) string
	Join(paths ...string) string
}
