package execute

import (
	"fmt"
	"sync/atomic"

	"github.com/jeffrom/polyester/compiler"
	"github.com/jeffrom/polyester/operator"
)

type execPool struct {
	workers []*runWorker
	out     chan *PlanResult
	n       int64
	stopC   chan struct{}
	allDone chan *Result

	planStartC     chan *compiler.Plan
	planDoneC      chan *PlanResult
	unblockWrkPoll chan struct{}
}

func newExecPool(conc int) *execPool {
	out := make(chan *PlanResult)
	workers := make([]*runWorker, conc)
	for i := 0; i < conc; i++ {
		workers[i] = newRunWorker(out, 0)
	}
	return &execPool{
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
	ep.planStartC <- plan
}

func (ep *execPool) enqueueOnePlan(plan *compiler.Plan, pc *planCache) {
	if pc.seen[plan.Name] {
		fmt.Println("enqueueOnePlan: already seen:", plan.Name)
		return
	}
	pc.seen[plan.Name] = true
	atomic.AddInt64(&ep.n, 1)

	fmt.Printf("enqueueOnePlan: enqueueing plan %s (%p) (%d workers)\n", plan.Name, plan, len(ep.workers))
	for {
		// try each worker repeatedly
		for _, wrk := range ep.workers {
			fmt.Printf("enqueueOnePlan: checking worker %p\n", wrk)
			select {
			case wrk.in <- plan:
				ep.planStartC <- plan
				fmt.Printf("enqueueOnePlan: enqueued %s (%p)\n", plan.Name, plan)
				return
			default:
			}
		}
		fmt.Println("no available goroutines found, waiting for more")
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
	fmt.Println("wait done")
	return res, nil
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
			fmt.Printf("feeder <-plan: %+v\n", plan)
			ep.runPending(pc)
			ep.feedPlan(plan, pc)
		case res := <-ep.planDoneC:
			fmt.Printf("feeder <-res: %+v\n", res)
			ep.finishPlan(res, pc)
		}
	}
}

func (ep *execPool) runPending(pc *planCache) {
	fmt.Println("pending")
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
	fmt.Println("feedPlan deps:", plan.Name, len(plan.Dependencies))
	for _, dep := range plan.Dependencies {
		fmt.Printf("feedPlan dep %s (%p)\n", dep.Name, dep)
		ep.feedPlan(dep, pc)
	}
	fmt.Println("feedPlan plans:", plan.Name, len(plan.Plans))
	for _, sp := range plan.Plans {
		fmt.Printf("feedPlan plan %s (%p)\n", sp.Name, sp)
		ep.feedPlan(sp, pc)
	}

	fmt.Println("feedPlan maino", plan.Name)
	if len(plan.RealOps()) > 0 && pc.areDepsComplete(plan) {
		fmt.Printf("feedPlan main %s (%p)\n", plan.Name, plan)
		ep.enqueueOnePlan(plan, pc)
	} else {
		pc.pending[plan] = true
	}
	fmt.Printf("feedPlan %s (%p): deps: %d, plans: %d, operations: %d\n", plan.Name, plan, len(plan.Dependencies), len(plan.Plans), len(plan.Operations))
}

func (ep *execPool) finishPlan(res *PlanResult, pc *planCache) {
	fmt.Printf("finishPlan %s %p\n", res.Plan.Name, res.Plan)
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
			fmt.Printf("gatherer res: %+v\n", res)
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
				fmt.Println("gathererLoop all done!")
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
	// time.Sleep(200 * time.Millisecond)
	return &PlanResult{Plan: plan}, nil
}
