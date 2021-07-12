package operator

import (
	"bufio"
	"encoding/json"
	"io"
	"os"
	"sort"

	"github.com/jeffrom/polyester/operator/opfs"
)

type State struct {
	Entries []StateEntry `json:"entries"`
}

// func NewStateFromPath(p string) (State, error) {
// }

func (s State) WriteFile(p string) error {
	f, err := os.Create(p)
	if err != nil {
		return err
	}
	defer f.Close()
	_, err = s.WriteTo(f)
	return err
}

func (s State) WriteTo(w io.Writer) (int64, error) {
	bw := bufio.NewWriter(w)
	b, err := json.Marshal(s)
	if err != nil {
		return 0, err
	}
	if _, err := bw.Write(b); err != nil {
		return int64(bw.Size()), err
	}
	return int64(bw.Size()), bw.Flush()
}

func (s State) Append(next ...StateEntry) State {
	entries := append(s.Entries, next...)
	return State{Entries: entries}
}

func (s State) Empty() bool { return len(s.Entries) == 0 }

func (s State) Changed(other State) bool {
	if len(s.Entries) != len(other.Entries) {
		return true
	}
	sort.Sort(stateEntries(s.Entries))
	sort.Sort(stateEntries(other.Entries))

	for i, ent := range s.Entries {
		oent := other.Entries[i]
		if ent.Name != oent.Name {
			return true
		}

		if (ent.File == nil) != (oent.File == nil) {
			return true
		}
		if ent.File != nil {
			sf, of := ent.File, oent.File
			if (sf.Info == nil) != (of.Info == nil) {
				return true
			}
			if sf.Info != nil &&
				sf.Info.IsDir() != of.Info.IsDir() ||
				sf.Info.Mode().Perm() != of.Info.Mode().Perm() ||
				!sf.Info.ModTime().Equal(of.Info.ModTime()) {
				return true
			}
		}
	}

	return false
}

type StateEntry struct {
	Name string               `json:"name"`
	File *opfs.StateFileEntry `json:"file,omitempty"`
	KV   map[string]string    `json:"kv,omitempty"`
}

type stateEntries []StateEntry

func (se stateEntries) Len() int {
	return len(se)
}

func (se stateEntries) Swap(i, j int) {
	se[i], se[j] = se[j], se[i]
}

func (se stateEntries) Less(i, j int) bool {
	return se[i].Name < se[j].Name
}
