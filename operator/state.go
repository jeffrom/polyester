package operator

import (
	"io/fs"
	"sort"
)

type State struct {
	entries []stateEntry
}

func (s State) Append(next ...stateEntry) State {
	entries := append(s.entries, next...)
	return State{entries: entries}
}

func (s State) Changed(other State) bool {
	if len(s.entries) != len(other.entries) {
		return true
	}
	sort.SortStable(stateEntries(s.entries))
	sort.SortStable(stateEntries(other.entries))

	for i, ent := range s.entries {
		oent := other[i]
		if ent.name != oent.name {
			return true
		}
	}

	return false
}

type stateEntry struct {
	name string
	abs  string
	f    fs.File
	info fs.FileInfo
}

type stateEntries []stateEntry

func (se stateEntries) Len() int {
	return len(se)
}

func (se stateEntries) Swap(i, j int) {
	se[i], se[j] = se[j], se[i]
}

func (se stateEntries) Less(i, j int) bool {
	return se[i].name < se[j].name
}
