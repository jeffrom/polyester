// Package templates contains a template manager.
package templates

import (
	"fmt"
	"io/fs"
	"path/filepath"
	"strings"
	"text/template"
)

// Templates is a collection of templates loaded from a plan directory, any of
// which can be rendered and import others.
type Templates struct {
	path      string
	dataPaths []string
	tmpl      *template.Template
}

func New(p string, dataPaths []string) *Templates {
	return &Templates{
		path:      p,
		dataPaths: dataPaths,
	}
}

func (t *Templates) Load() error {
	tmpl := template.New("plan")

	walkFn := func(p string, d fs.FileInfo, perr error) error {
		if perr != nil {
			return perr
		}
		if d.IsDir() {
			return nil
		}
		if !strings.Contains(p, "/templates") {
			return nil
		}
		fmt.Println("Load:", p)
		return nil
	}
	if err := filepath.Walk(t.path, walkFn); err != nil {
		return err
	}
	t.tmpl = tmpl
	return nil
}
