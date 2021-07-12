package planner

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"

	"github.com/jeffrom/polyester/operator"
)

type Plan struct {
	Operations []operator.Interface
}

func ReadFile(p string) (*Plan, error) {
	f, err := os.Open(p)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var ops []operator.Interface
	buf := &bytes.Buffer{}
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		line := sc.Bytes()
		isSep := bytes.Equal(bytes.TrimSpace(line), []byte("---"))
		if isSep {
			op, err := opFromBuf(buf)
			if err != nil {
				return nil, fmt.Errorf("failed to extract operation: %w", err)
			}
			if op != nil {
				ops = append(ops, op)
			}
			continue
		}

		if len(bytes.TrimSpace(line)) == 0 {
			continue
		}
		if _, err := buf.Write(line); err != nil {
			return nil, err
		}
		// fmt.Printf("raw line (total: %d): %s\n", buf.Len(), string(line))
	}
	if err := sc.Err(); err != nil {
		return nil, err
	}
	op, err := opFromBuf(buf)
	if err != nil {
		return nil, fmt.Errorf("failed to extract last operator: %w", err)
	}
	if op != nil {
		ops = append(ops, op)
	}
	return &Plan{Operations: ops}, nil
}

func opFromBuf(buf *bytes.Buffer) (operator.Interface, error) {
	defer buf.Reset()
	b := buf.Bytes()
	if len(b) == 0 {
		return nil, nil
	}
	entry := &operator.PlanEntry{}
	if err := yaml.Unmarshal(b, entry); err != nil {
		return nil, err
	}
	opc, ok := allOps[entry.Name]
	if !ok {
		return nil, fmt.Errorf("did not find operation %q", entry.Name)
	}
	op := opc()
	opData := op.Info().Data()
	opData.Command.Target = entry.Args
	return op, nil
}

func AppendPlan(ctx context.Context, planFile string, info operator.Info, cobraCmd *cobra.Command, args []string) error {
	cmd := info.Data().Command
	if err := cmd.ParseFlags(args); err != nil {
		return err
	}

	if cmd.ApplyArgs != nil {
		if err := cmd.ApplyArgs(cobraCmd, args, cmd.Target); err != nil {
			return err
		}
	}

	// fmt.Printf("%s: args %+v\n", info.Name(), cmd.Target)
	return appendToFile(planFile, info)
}

func appendToFile(file string, info operator.Info) error {
	if _, err := os.Stat(file); err != nil {
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}

	f, err := os.OpenFile(file, os.O_WRONLY|os.O_CREATE|os.O_APPEND, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	return info.Data().Encode(f)
}
