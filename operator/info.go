package operator

import "io"

type Info interface {
	Data() *InfoData
	TextSummary(w io.Writer) error
}

type InfoData struct {
	Description string `json:"description,omitempty"`
}
