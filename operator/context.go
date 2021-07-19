package operator

import (
	"context"
	"io/fs"

	"github.com/jeffrom/polyester/operator/opfs"
	"github.com/jeffrom/polyester/operator/templates"
)

type ctxKey string

var gotStateKey = ctxKey("gotState")

type Context struct {
	context.Context

	Opts      interface{}
	FS        FS
	PlanDir   opfs.PlanDir
	Templates *templates.Templates
}

func NewContext(ctx context.Context, fs FS, planDir opfs.PlanDir, tmpl *templates.Templates) Context {
	return Context{
		Context:   ctx,
		FS:        fs,
		PlanDir:   planDir,
		Templates: tmpl,
	}
}

func (c Context) WithValue(key, val interface{}) Context {
	ctx := context.WithValue(c.Context, key, val)
	return Context{
		Context:   ctx,
		Opts:      c.Opts,
		FS:        c.FS,
		PlanDir:   c.PlanDir,
		Templates: c.Templates,
	}
}

func (c Context) Value(key interface{}) interface{} {
	return c.Context.Value(key)
}

func (c Context) WithSubplan(spdir string) Context {
	return Context{
		Context:   c.Context,
		Opts:      c.Opts,
		FS:        c.FS,
		PlanDir:   c.PlanDir.WithSubplan(spdir),
		Templates: c.Templates,
	}
}

func (c Context) WithGotState(gotState bool) Context {
	ctx := context.WithValue(c.Context, gotStateKey, gotState)
	return Context{
		Context:   ctx,
		Opts:      c.Opts,
		FS:        c.FS,
		PlanDir:   c.PlanDir,
		Templates: c.Templates,
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
