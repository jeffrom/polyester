package execute

import (
	"fmt"
	"sync/atomic"
	"time"

	"github.com/jeffrom/polyester/compiler"
	"github.com/jeffrom/polyester/operator"
)

type execPool struct {
	workers []*runWorker
	n       int64
}

func newExecPool(out chan<- *PlanResult, conc int) *execPool {
	workers := make([]*runWorker, conc)
	for i := 0; i < conc; i++ {
		workers[i] = newRunWorker(out, 0)
	}
	return &execPool{
		workers: workers,
	}
}

func (ep *execPool) addPlan(plan *compiler.Plan) {
	for {
		for _, wrk := range ep.workers {
			select {
			case wrk.in <- plan:
				atomic.AddInt64(&ep.n, 1)
				return
			default:
				continue
			}
		}
		fmt.Println("no available workers found??")
		time.Sleep(500 * time.Millisecond)
	}
}

func (ep *execPool) start(octx operator.Context, opts Opts) {
	// - one goroutine to keep track of which plans to run next, add plans when
	// their dependencies are done etc.
	// - one to collect results

	for _, wrk := range ep.workers {
		wrk.start(octx, opts)
	}
}

func (ep *execPool) wait() (*Result, error) {
	return nil, nil
}

// runWorker executes plans in a goroutine, one at a time. The goroutine can
// persist the entirety of a polyester apply. N instances of runWorker should
// be instantiated in a pool.
type runWorker struct {
	in    chan *compiler.Plan
	out   chan<- *PlanResult
	stopC chan struct{}
}

func newRunWorker(out chan<- *PlanResult, queueSize int) *runWorker {
	return &runWorker{
		in:    make(chan *compiler.Plan, queueSize),
		out:   out,
		stopC: make(chan struct{}),
	}
}

func (wrk *runWorker) start(octx operator.Context, opts Opts) {
	go wrk.loop(octx, opts)
}

func (wrk *runWorker) stop() error {
	close(wrk.stopC)
	return nil
}

func (wrk *runWorker) loop(octx operator.Context, opts Opts) {
Cleanup:
	for {
		select {
		case <-octx.Context.Done():
			break Cleanup
		case <-wrk.stopC:
			break Cleanup

		case plan := <-wrk.in:
			fmt.Printf("starting: %s\n", plan.Name)
			planRes, err := wrk.executePlan(octx, opts)
			if err != nil {
				if planRes == nil {
					planRes = &PlanResult{Name: plan.Name, Error: err}
				} else {
					planRes.Error = err
				}
			}
			wrk.out <- planRes
		}
	}
}

func (wrk *runWorker) add(plan *compiler.Plan) { wrk.in <- plan }

func (wrk *runWorker) executePlan(octx operator.Context, opts Opts) (*PlanResult, error) {
	// - need to set subplan on octx here
	return nil, nil
}
