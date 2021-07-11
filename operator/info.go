package operator

import (
	"io"

	"github.com/spf13/cobra"
)

type Info interface {
	Data() *InfoData
	TextSummary(w io.Writer) error
}

type InfoData struct {
	Command *Command `json:"options,omitempty"`
}

func (id *InfoData) Data() *InfoData { return id }
func (id *InfoData) TextSummary(w io.Writer) error {
	_, err := w.Write([]byte(id.Command.UsageString()))
	return err
}

type CommandArgFunc func(cmd *cobra.Command, args []string, target interface{}) error

type Command struct {
	*cobra.Command
	Args   CommandArgFunc
	Target interface{}
}
