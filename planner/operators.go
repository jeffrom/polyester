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
		func() operator.Interface { return fileop.Touch{} },
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
