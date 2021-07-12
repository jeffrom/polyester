package planner

import (
	"sync"

	"github.com/jeffrom/polyester/operator"
	"github.com/jeffrom/polyester/operator/fileop"
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
		func() operator.Interface { return fileop.Touch{Args: &fileop.TouchOpts{}} },
	}
}

func Operators() []operator.Interface {
	ops := opCreators()
	res := make([]operator.Interface, len(ops))
	for i, opc := range ops {
		res[i] = opc()
	}
	return res
}

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
