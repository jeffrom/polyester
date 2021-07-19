package planner

import (
	"context"

	"github.com/jeffrom/polyester/operator/templates"
)

func (r *Planner) setupTemplates(ctx context.Context) (*templates.Templates, error) {
	tmpl := templates.New(r.planDir)
	if err := tmpl.Load(); err != nil {
		return nil, err
	}
	return tmpl, nil
}
