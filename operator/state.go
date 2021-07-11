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
	sort.Sort(stateEntries(s.entries))
	sort.Sort(stateEntries(other.entries))

	for i, ent := range s.entries {
		oent := other.entries[i]
		if ent.name != oent.name {
			return true
		}

		if (ent.file == nil) != (oent.file == nil) {
			return true
		}
		if ent.file != nil {
			sf, of := ent.file, oent.file
			if sf.abs != of.abs ||
				sf.info.IsDir() != of.info.IsDir() ||
				sf.info.Mode().Perm() != of.info.Mode().Perm() {
				return true
			}
		}
	}

	return false
}

type stateEntry struct {
	name string
	file *stateFileEntry
}

type stateFileEntry struct {
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
