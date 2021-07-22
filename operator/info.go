package operator

import (
	"encoding/json"
	"io"

	"github.com/ghodss/yaml"
	"github.com/spf13/cobra"
)

type Info interface {
	Name() string
	Data() *InfoData
	TextSummary(w io.Writer) error
}

type InfoData struct {
	OpName  string   `json:"name"`
	Command *Command `json:"command"`
}

func (id *InfoData) Copy() *InfoData {
	cp := *id
	cp.Command = id.Command
	return &cp
}

func (id *InfoData) Data() *InfoData { return id }
func (id *InfoData) Name() string    { return id.OpName }
func (id *InfoData) TextSummary(w io.Writer) error {
	_, err := w.Write([]byte(id.Command.UsageString()))
	return err
}

func (id *InfoData) Encode(w io.Writer) error {
	targetb, err := json.Marshal(id.Command.Target)
	if err != nil {
		return err
	}
	pe := &PlanEntry{
		Name: id.Name(),
		Args: targetb,
	}
	b, err := yaml.Marshal(pe)
	if err != nil {
		return err
	}

	if _, err := w.Write(b); err != nil {
		return err
	}
	if _, err := w.Write([]byte("---\n")); err != nil {
		return err
	}
	return nil
}

type CommandArgFunc func(cmd *cobra.Command, args []string, target interface{}) error

type Command struct {
	*cobra.Command
	ApplyArgs CommandArgFunc
	Target    interface{}
}

type PlanEntry struct {
	Name string          `json:"name"`
	Args json.RawMessage `json:"args,omitempty"`
}
