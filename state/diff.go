package state

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/sergi/go-diff/diffmatchpatch"
)

func (s State) Diff(w io.Writer, other State) error {
	ab, err := json.MarshalIndent(s.Entries, "", "  ")
	if err != nil {
		return err
	}
	bb, err := json.MarshalIndent(other.Entries, "", "  ")
	if err != nil {
		return err
	}

	dmp := diffmatchpatch.New()
	diffs := computeDiffs(string(ab), string(bb))
	_, err = fmt.Fprint(w, dmp.DiffPrettyText(diffs))
	return err
}

func computeDiffs(a, b string) []diffmatchpatch.Diff {
	dmp := diffmatchpatch.New()
	diffs := dmp.DiffMain(a, b, true)
	diffs = dmp.DiffCleanupSemantic(diffs)
	diffs = dmp.DiffCleanupEfficiency(diffs)
	return diffs
}
