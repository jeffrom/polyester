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
	out     chan *PlanResult
	n       int64
	stopC   chan struct{}

	planStartC chan *compiler.Plan
	planDoneC  chan *PlanResult
}

func newExecPool(conc int) *execPool {
	out := make(chan *PlanResult)
	workers := make([]*runWorker, conc)
	for i := 0; i < conc; i++ {
		workers[i] = newRunWorker(out, 0)
	}
	return &execPool{
		out:        out,
		workers:    workers,
		stopC:      make(chan struct{}),
		planStartC: make(chan *compiler.Plan),
		planDoneC:  make(chan *PlanResult),
	}
}

// add should be called to run a plan concurrently. The very top level
// manifest should be passed to this function, which will resolve dependencies
// and enqueue plan executions in the right order.
func (ep *execPool) add(plan *compiler.Plan) {
	ep.planStartC <- plan
}

func (ep *execPool) enqueueOnePlan(plan *compiler.Plan, pc *planCache) {
	if pc.seen[plan.Name] {
		fmt.Println("enqueueOnePlan: already seen:", plan.Name)
		return
	}

	fmt.Printf("enqueueOnePlan: enqueueing plan %s (%p)\n", plan.Name, plan)
	for {
		// try each worker repeatedly
		for _, wrk := range ep.workers {
			fmt.Printf("enqueueOnePlan: checking worker %p\n", wrk)
			select {
			case wrk.in <- plan:
				atomic.AddInt64(&ep.n, 1)
				ep.planStartC <- plan
				pc.seen[plan.Name] = true
				fmt.Printf("enqueueOnePlan: enqueued %s (%p)\n", plan.Name, plan)
				return
			default:
				continue
			}
		}
		fmt.Println("no available workers goroutines found??")
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

	go ep.gathererLoop(octx, opts)
	go ep.feederLoop(octx, opts)
}

func (ep *execPool) wait() (*Result, error) {
	// XXX lol
	time.Sleep(1 * time.Second)
	return &Result{}, nil
}

func (ep *execPool) feederLoop(octx operator.Context, opts Opts) {
	pc := newPlanCache()
	for {
		select {
		case <-octx.Context.Done():
			return
		case <-ep.stopC:
			return
		case plan := <-ep.planStartC:
			fmt.Printf("feeder <-plan: %+v\n", plan)
			ep.feedPlan(plan, pc)
		case res := <-ep.planDoneC:
			fmt.Printf("feeder <-res: %+v\n", res)
			ep.finishPlan(res, pc)
		}
	}
}

// feedPlan resolves the plan and enqueues its subplans one at a time, followed
// by its own operations, if any
func (ep *execPool) feedPlan(plan *compiler.Plan, pc *planCache) {
	fmt.Println("pending")
	for pl := range pc.pending {
		if pc.areDepsComplete(pl) {
			ep.enqueueOnePlan(pl, pc)
			delete(pc.pending, pl)
		}
	}

	for _, dep := range plan.Dependencies {
		fmt.Printf("feedPlan dep %s (%p)", dep.Name, dep)
		if pc.areDepsComplete(dep) {
			ep.enqueueOnePlan(dep, pc)
		} else {
			pc.pending[dep] = true
		}
	}
	fmt.Println("plans")
	for _, sp := range plan.Plans {
		fmt.Printf("feedPlan plan %s (%p)", sp.Name, sp)
		if pc.areDepsComplete(sp) {
			ep.enqueueOnePlan(sp, pc)
		} else {
			pc.pending[sp] = true
		}
	}

	fmt.Println("maino")
	if pc.areDepsComplete(plan) {
		fmt.Printf("feedPlan main %s (%p)", plan.Name, plan)
		ep.enqueueOnePlan(plan, pc)
	} else {
		pc.pending[plan] = true
	}
	fmt.Printf("feedPlan %s (%p): deps: %d, plans: %d, operations: %d\n", plan.Name, plan, len(plan.Dependencies), len(plan.Plans), len(plan.Operations))
}

func (ep *execPool) finishPlan(res *PlanResult, pc *planCache) {
	// TODO lookup any dependents that can run now that this plan has completed and
	// enqueue them, then remove them from the pending map

	pc.done[res.Plan] = true
	ep.feedPlan(res.Plan, pc)
}

func (ep *execPool) gathererLoop(octx operator.Context, opts Opts) {
	for {
		select {
		case <-octx.Context.Done():
			return
		case <-ep.stopC:
			return
		case res := <-ep.out:
			atomic.AddInt64(&ep.n, -1)
			fmt.Printf("gatherer res: %+v\n", res)
			ep.planDoneC <- res
		}
	}
}

type planCache struct {
	seen    map[string]bool
	pending map[*compiler.Plan]bool
	done    map[*compiler.Plan]bool
}

func newPlanCache() *planCache {
	return &planCache{
		seen:    make(map[string]bool),
		pending: make(map[*compiler.Plan]bool),
		done:    make(map[*compiler.Plan]bool),
	}
}

func (pc *planCache) areDepsComplete(plan *compiler.Plan) bool {
	for _, dep := range plan.Dependencies {
		if !pc.done[dep] {
			return false
		}
	}
	return true
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
			fmt.Printf("runWorker %p: starting: %s\n", wrk, plan.Name)
			planRes, err := wrk.executePlan(octx, opts, plan)
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

func (wrk *runWorker) executePlan(octx operator.Context, opts Opts, plan *compiler.Plan) (*PlanResult, error) {
	fmt.Printf("runWorker %p: executePlan %s\n", wrk, plan.Name)
	// - need to set subplan on octx here
	return &PlanResult{Plan: plan}, nil
}
