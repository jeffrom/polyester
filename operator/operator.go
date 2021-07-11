// Package operator is an interface for defining arbitrary operators,
// which gather state and make changes to an environment.
package operator

type Interface interface {
	Info() Info
	GetState(octx Context) (State, error)
	Run(octx Context) error
}
