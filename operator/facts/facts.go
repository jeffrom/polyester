// Package facts manages "facts", which are data gathered from the environment
// of the agent.
package facts

import (
	"context"
	"sync"

	"github.com/zcalusic/sysinfo"
)

var (
	si     sysinfo.SysInfo
	siOnce sync.Once
)

type Facts struct {
	System sysinfo.SysInfo `json:"system"`
}

func Gather(ctx context.Context) (Facts, error) {
	siOnce.Do(si.GetSysInfo)
	return Facts{System: si}, nil
}
