// Package templates contains a template manager.
package templates

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"text/template"

	"github.com/jeffrom/polyester/operator/facts"
)

// Templates is a collection of templates loaded from a plan directory, any of
// which can be rendered and import others.
type Templates struct {
	tmpl *template.Template
	path string
}

func New(p string) *Templates {
	return &Templates{
		path: p,
	}
}

func (t *Templates) Load() error {
	tmpl := template.Must(template.New("plan").Parse(""))

	var tmplPaths []string
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
		tmplPaths = append(tmplPaths, p)
		return nil
	}
	if err := filepath.Walk(t.path, walkFn); err != nil {
		return err
	}

	for _, p := range tmplPaths {
		b, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		tp := convertTemplatePath(t.path, p)
		// fmt.Println("Loading:", tp, p)
		_, err = tmpl.New(tp).Parse(string(b))
		if err != nil {
			return err
		}
	}
	// fmt.Println("donezo", tmpl.Templates())
	// for _, t := range tmpl.Templates() {
	// 	fmt.Println(t.Name(), t.Mode)
	// }
	t.tmpl = tmpl
	return nil
}

func (t *Templates) ExecuteForOp(w io.Writer, name string, data Data) error {
	tmpl := t.tmpl.Lookup(name)
	if tmpl == nil {
		return fmt.Errorf("templates: could not find %q", name)
	}
	facts, err := facts.Gather()
	if err != nil {
		return err
	}
	data.Facts = facts
	return tmpl.Execute(w, data)
}

const sep = string(filepath.Separator)

func convertTemplatePath(root, p string) string {
	withoutPlanDir := strings.TrimPrefix(p, root+sep)
	if strings.HasPrefix(withoutPlanDir, "plans"+sep) {
		parts := strings.Split(withoutPlanDir, sep)
		return filepath.Join(append([]string{parts[1]}, parts[3:]...)...)
	}
	res := strings.TrimPrefix(withoutPlanDir, "templates"+sep)
	return res
}
