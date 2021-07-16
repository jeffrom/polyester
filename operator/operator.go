// Package operator is an interface for defining arbitrary operators,
// which gather state and make changes to an environment.
package operator

type Interface interface {
	Info() Info
	GetState(octx Context) (State, error)
	Run(octx Context) error
}

// Validater can be implemented by operators to validate arguments before
// execution. If implemented, Validate can be run multiple times. If evaluated
// is false, the arguments shell script variables are not expanded -- ie if the
// argument in the shell script is "$mydir/myfile", that is what the command
// target args will be on the operation. If true, the arguments shell operators
// are expanded.
type Validater interface {
	Validate(octx Context, targ interface{}, evaluated bool) error
}

// DesiredStater can be implemented by operators to make it possible to skip
// operator.Run, mark the state unchanged, and still write the resulting state.
// When this is implemented, the planner will run it before executing the
// operator, and if the requested state (as determined by DesiredState())
// matches the current state, plan application will continue as if the operator
// was executed.
type DesiredStater interface {
	DesiredState(octx Context) (State, error)
}
