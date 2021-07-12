package operator

import (
	"context"
	"io/fs"
)

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

type FS interface {
	fs.StatFS
	fs.GlobFS
	fs.ReadDirFS
	fs.ReadFileFS
	Abs(name string) string
	Join(paths ...string) string
}
