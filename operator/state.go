package operator

import (
	"io/fs"
	"sort"
)

type State struct {
	entries []StateEntry
}

func (s State) Append(next ...StateEntry) State {
	entries := append(s.entries, next...)
	return State{entries: entries}
}

func (s State) Changed(other State) bool {
	if len(s.entries) != len(other.entries) {
		return true
	}
	sort.Sort(stateEntries(s.entries))
	sort.Sort(stateEntries(other.entries))

	for i, ent := range s.entries {
		oent := other.entries[i]
		if ent.Name != oent.Name {
			return true
		}

		if (ent.File == nil) != (oent.File == nil) {
			return true
		}
		if ent.File != nil {
			sf, of := ent.File, oent.File
			if sf.Abs != of.Abs ||
				sf.Info.IsDir() != of.Info.IsDir() ||
				sf.Info.Mode().Perm() != of.Info.Mode().Perm() {
				return true
			}
		}
	}

	return false
}

type StateEntry struct {
	Name string
	File *StateFileEntry
}

type StateFileEntry struct {
	fs.File
	Abs  string
	Info fs.FileInfo
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
