// Package operator is an interface for defining arbitrary operators,
// which gather state and make changes to an environment.
package operator

import (
	"context"
)

type Interface interface {
	Name() string
	Info() Info
	GetState(ctx context.Context) (State, error)
	Run(ctx context.Context) error
}
