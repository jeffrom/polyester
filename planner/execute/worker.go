package execute

import (
	"fmt"
	"sync/atomic"

	"github.com/jeffrom/polyester/compiler"
	"github.com/jeffrom/polyester/operator"
	"github.com/jeffrom/polyester/stdio"
)

type execPool struct {
	std     stdio.StdIO
	workers []*runWorker
	out     chan *PlanResult
	n       int64
	stopC   chan struct{}
	allDone chan *Result

	planStartC     chan *compiler.Plan
	planDoneC      chan *PlanResult
	unblockWrkPoll chan struct{}
}

func newExecPool(conc int, std stdio.StdIO) *execPool {
	out := make(chan *PlanResult)
	workers := make([]*runWorker, conc)
	for i := 0; i < conc; i++ {
		workers[i] = newRunWorker(out, 1)
	}
	return &execPool{
		std:            std,
		out:            out,
		workers:        workers,
		stopC:          make(chan struct{}),
		allDone:        make(chan *Result),
		planStartC:     make(chan *compiler.Plan, 100),
		planDoneC:      make(chan *PlanResult, 100),
		unblockWrkPoll: make(chan struct{}),
	}
}

// add should be called to run a plan concurrently. The very top level
// manifest should be passed to this function, which will resolve dependencies
// and enqueue plan executions in the right order.
func (ep *execPool) add(plan *compiler.Plan) {
	ep.std.Debugf("execPool: add plan: %s", plan.Name)
	ep.planStartC <- plan
}

func (ep *execPool) enqueueOnePlan(plan *compiler.Plan, pc *planCache) {
	if pc.seen[plan.Name] {
		ep.std.Debug("enqueueOnePlan: already seen:", plan.Name)
		return
	}
	pc.seen[plan.Name] = true
	atomic.AddInt64(&ep.n, 1)

	ep.std.Debug("enqueueOnePlan: enqueueing plan %s (%p) (%d workers)\n", plan.Name, plan, len(ep.workers))
	for {
		// try each worker repeatedly
		for _, wrk := range ep.workers {
			ep.std.Debugf("enqueueOnePlan: checking worker %p", wrk)
			select {
			case wrk.in <- plan:
				ep.planStartC <- plan
				ep.std.Debugf("enqueueOnePlan: enqueued %s (%p)", plan.Name, plan)
				return
			default:
				ep.std.Debugf("enqueueOnePlan: worker %p wasn't available", wrk)
			}
		}
		ep.std.Debug("execPool: no available goroutines found, waiting for more")
		<-ep.unblockWrkPoll
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
	res := <-ep.allDone
	ep.std.Infof("wait done")
	return res, res.Err()
}

func (ep *execPool) stop() {
	for _, wrk := range ep.workers {
		close(wrk.stopC)
	}
	close(ep.stopC)
}

func (ep *execPool) feederLoop(octx operator.Context, opts Opts) {
	pc := newPlanCache()
	for {
		ep.runPending(pc)
		select {
		case <-octx.Context.Done():
			return
		case <-ep.stopC:
			return
		case plan := <-ep.planStartC:
			ep.std.Debugf("feeder <-plan: %+v", plan)
			ep.runPending(pc)
			ep.feedPlan(plan, pc)
		case res := <-ep.planDoneC:
			ep.std.Debugf("feeder <-res: %+v", res)
			ep.finishPlan(res, pc)
		}
	}
}

func (ep *execPool) runPending(pc *planCache) {
	ep.std.Debug("pending")
	for pl := range pc.pending {
		if pc.areDepsComplete(pl) {
			ep.enqueueOnePlan(pl, pc)
			delete(pc.pending, pl)
		}
	}
}

// feedPlan resolves the plan and enqueues its subplans one at a time, followed
// by its own operations, if any
func (ep *execPool) feedPlan(plan *compiler.Plan, pc *planCache) {
	ep.std.Debug("feedPlan deps:", plan.Name, len(plan.Dependencies))
	for _, dep := range plan.Dependencies {
		ep.std.Debugf("feedPlan dep %s (%p)", dep.Name, dep)
		ep.feedPlan(dep, pc)
	}
	ep.std.Debug("feedPlan plans:", plan.Name, len(plan.Plans))
	for _, sp := range plan.Plans {
		ep.std.Debugf("feedPlan plan %s (%p)", sp.Name, sp)
		ep.feedPlan(sp, pc)
	}

	ep.std.Debug("feedPlan main:", plan.Name)
	if len(plan.RealOps()) > 0 && pc.areDepsComplete(plan) {
		ep.std.Debugf("feedPlan main %s (%p)", plan.Name, plan)
		ep.enqueueOnePlan(plan, pc)
	} else {
		pc.pending[plan] = true
	}
	ep.std.Debugf("feedPlan %s (%p): deps: %d, plans: %d, operations: %d", plan.Name, plan, len(plan.Dependencies), len(plan.Plans), len(plan.Operations))
}

func (ep *execPool) finishPlan(res *PlanResult, pc *planCache) {
	if res == nil {
		ep.std.Debug("plan finished with nil result")
		return
	}
	if res.Plan == nil {
		ep.std.Debugf("plan finished w/out result.plan. error: %v", res.Error)
		return
	}
	ep.std.Debugf("finishPlan %s %p", res.Plan.Name, res.Plan)
	pc.done[res.Plan] = true
	ep.feedPlan(res.Plan, pc)
}

func (ep *execPool) gathererLoop(octx operator.Context, opts Opts) {
	var unsortedResults []*PlanResult
	for {
		select {
		case <-octx.Context.Done():
			return
		case <-ep.stopC:
			return
		case res := <-ep.out:
			allDone := false
			if atomic.AddInt64(&ep.n, -1) == 0 {
				allDone = true
			}
			ep.std.Debugf("gatherer res: %+v", res)
			unsortedResults = append(unsortedResults, res)
			select {
			case ep.planDoneC <- res:
				select {
				case ep.unblockWrkPoll <- struct{}{}:
				default:
				}
			default:
				panic("unable to notify finish of: " + res.Plan.Name)
			}

			if allDone {
				ep.allDone <- &Result{Plans: unsortedResults}
				ep.std.Debug("gathererLoop all done!")
				return
			}
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
	std := stdio.FromContext(octx.Context)
	std.Debugf("worker %p start", wrk)
	go wrk.loop(octx, opts)
}

// func (wrk *runWorker) stop() error {
// 	close(wrk.stopC)
// 	return nil
// }

func (wrk *runWorker) loop(octx operator.Context, opts Opts) {
	std := stdio.FromContext(octx.Context)
Cleanup:
	for {
		select {
		case <-octx.Context.Done():
			break Cleanup
		case <-wrk.stopC:
			break Cleanup

		case plan := <-wrk.in:
			std.Debugf("runWorker %p: starting: %s", wrk, plan.Name)
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
	std.Debugf("worker %p loop done", wrk)
}

func (wrk *runWorker) executePlan(octx operator.Context, opts Opts, plan *compiler.Plan) (*PlanResult, error) {
	std := stdio.FromContext(octx.Context).WithScope(plan.Name, fmt.Sprintf("%p", wrk))
	std.Debugf("runWorker %p: executePlan %s", wrk, plan.Name)
	return executePlan(octx, std, opts, plan)
}
