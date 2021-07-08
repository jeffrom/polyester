// Package fact manages "facts", which are data gathered from the environment
// of the agent.
package fact

import "context"

type Facts struct {
}

func Gather(ctx context.Context) (*Facts, error) {
	return nil, nil
}
