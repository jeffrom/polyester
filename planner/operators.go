package planner

import (
	"sync"

	"github.com/jeffrom/polyester/operator"
	"github.com/jeffrom/polyester/operator/fileop"
	"github.com/jeffrom/polyester/operator/gitop"
	"github.com/jeffrom/polyester/operator/pkgop"
	"github.com/jeffrom/polyester/operator/planop"
)

var (
	allOps      map[string]func() operator.Interface
	allOptsOnce = sync.Once{}
)

func setupAllOps() {
	allOps = make(map[string]func() operator.Interface)
	for _, opc := range opCreators() {
		op := opc()
		allOps[op.Info().Name()] = opc
	}
}

func opCreators() []func() operator.Interface {
	return []func() operator.Interface{
		func() operator.Interface { return planop.Plan{Args: &planop.PlanOpts{}} },
		func() operator.Interface { return planop.Dependency{Args: &planop.DependencyOpts{}} },
		func() operator.Interface { return fileop.Touch{Args: &fileop.TouchOpts{}} },
		func() operator.Interface { return gitop.Repo{Args: &gitop.RepoOpts{}} },
		func() operator.Interface { return pkgop.AptInstall{Args: &pkgop.AptInstallOpts{}} },
	}
}

// Operators returns a list of all available operators.
func Operators() []operator.Interface {
	ops := opCreators()
	res := make([]operator.Interface, len(ops))
	for i, opc := range ops {
		res[i] = opc()
	}
	return res
}

// operation is an implementation of operator.Interface that uses decoded Plan
// arguments instead of parsing them in-process.
type operation struct {
	op   operator.Interface
	data *operator.InfoData
}

func (op operation) Info() operator.Info { return op.data }
func (op operation) GetState(octx operator.Context) (operator.State, error) {
	return op.op.GetState(octx)
}
func (op operation) Run(octx operator.Context) error {
	return op.op.Run(octx)
}
