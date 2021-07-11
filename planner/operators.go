package planner

import (
	"github.com/jeffrom/polyester/operator"
	"github.com/jeffrom/polyester/operator/fileop"
)

func Operators() []operator.Interface {
	return []operator.Interface{
		fileop.Touch{},
	}
}
