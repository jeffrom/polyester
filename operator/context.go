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

type FS interface {
	fs.StatFS
	fs.GlobFS
	fs.ReadDirFS
	fs.ReadFileFS
}
