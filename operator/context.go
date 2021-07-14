package operator

import (
	"context"
	"io/fs"

	"github.com/jeffrom/polyester/operator/opfs"
)

type ctxKey string

var gotStateKey = ctxKey("gotState")

type Context struct {
	context.Context

	Opts    interface{}
	FS      FS
	PlanDir opfs.PlanDir
}

func NewContext(ctx context.Context, fs FS, planDir opfs.PlanDir) Context {
	return Context{
		Context: ctx,
		FS:      fs,
		PlanDir: planDir,
	}
}

func (c Context) WithValue(key, val interface{}) Context {
	ctx := context.WithValue(c.Context, key, val)
	return Context{
		Context: ctx,
		Opts:    c.Opts,
		FS:      c.FS,
		PlanDir: c.PlanDir,
	}
}

func (c Context) Value(key interface{}) interface{} {
	return c.Context.Value(key)
}

func (c Context) WithGotState(gotState bool) Context {
	ctx := context.WithValue(c.Context, gotStateKey, gotState)
	return Context{
		Context: ctx,
		Opts:    c.Opts,
		FS:      c.FS,
		PlanDir: c.PlanDir,
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
	Join(paths ...string) string
	// Abs(name string) string
}
