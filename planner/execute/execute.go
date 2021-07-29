// Package execute contains the logic to execute plans concurrently, taking
// into account dependencies and phases.
package execute

import (
	"io"

	"github.com/jeffrom/polyester/compiler"
	"github.com/jeffrom/polyester/operator"
	"github.com/jeffrom/polyester/state"
)

type Opts struct {
	Dryrun   bool
	DirRoot  string
	StateDir string
}

type ExecutionNode struct {
	plan  *compiler.Plan
	nodes []ExecutionNode
	thens []ExecutionNode
}

func New(plan *compiler.Plan) ExecutionNode { return ExecutionNode{plan: plan} }

func (en ExecutionNode) append(plan *compiler.Plan) ExecutionNode {
	en.nodes = append(en.nodes, ExecutionNode{plan: plan})
	return en
}

func (en ExecutionNode) then(plan *compiler.Plan) ExecutionNode {
	en.thens = append(en.thens, ExecutionNode{plan: plan})
	return en
}

func (en ExecutionNode) TextSummary(w io.Writer) error {
	return en.textSummary(w, en, 0)
}

func (en ExecutionNode) textSummary(w io.Writer, node ExecutionNode, depth int) error {
	// for _, node := range node.nodes {
	// 	io.WriteString(w, fmt.Sprintf("%s ∟ %s\n", strings.Repeat(" ", depth*2), node.plan.Name))
	// 	for _, then := range node.thens {
	// 		io.WriteString(w, fmt.Sprintf("%s ∟ %s\n", strings.Repeat(" ", (depth*2)+2), then.plan.Name))
	// 	}
	// }
	return nil
}

func (en ExecutionNode) GetState(octx operator.Context, opts Opts) ([]state.State, error) {
	return nil, nil
}

func (en ExecutionNode) Do(octx operator.Context, opts Opts) (*Result, error) {
	// for each top-level en.plan.Plans, run it concurrently.
	// everywhere in plan scripts where the "plan" or "dependency" operator is
	// called, the target plan will run concurrently, taking into account
	// dependencies.
	// operations run serially per-plan

	// safety checks:
	// - plan should only run once per apply run
	// - all of a plans dependencies must run before it is run

	// at the top level we need to keep track of the order, and put things back
	// together before returning the *Result.
	return nil, nil
}
