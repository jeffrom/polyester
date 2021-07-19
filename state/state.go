// Package state manages polyester states.
package state

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"
	"reflect"

	"github.com/jeffrom/polyester/operator/opfs"
)

type State struct {
	Entries []Entry `json:"entries"`
}

func New() State { return State{} }

func FromReader(r io.Reader) (State, error) {
	st := New()
	if err := json.NewDecoder(r).Decode(&st); err != nil {
		return st, err
	}
	return st, nil
}

func FromPath(p string) (State, error) {
	f, err := os.Open(p)
	if err != nil {
		return State{}, err
	}
	defer f.Close()
	return FromReader(bufio.NewReader(f))
}

func FromBytes(b []byte) (State, error) {
	return FromReader(bytes.NewReader(b))
}

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

func (s State) Append(next ...Entry) State {
	entries := append(s.Entries, next...)
	return State{Entries: entries}
}

func (s State) Source() State {
	var ents []Entry
	for _, ent := range s.Entries {
		if ent.Target {
			continue
		}
		ents = append(ents, ent)
	}
	return State{Entries: ents}
}

func (s State) Target() State {
	var ents []Entry
	for _, ent := range s.Entries {
		if !ent.Target {
			continue
		}
		ents = append(ents, ent)
	}
	return State{Entries: ents}
}

func (s State) Empty() bool {
	return len(s.Entries) == 0
}

func (s State) Changed(other State) bool {
	ents, oents := s.Entries, other.Entries
	if len(ents) != len(oents) {
		// fmt.Println("changed bc diff size")
		return true
	}
	// sort.Sort(stateEntries(ents))
	// sort.Sort(stateEntries(oents))

	for i, ent := range ents {
		oent := oents[i]
		if ent.Changed(oent) {
			return true
		}
	}

	return false
}

type Entry struct {
	Name   string                 `json:"name"`
	File   *opfs.StateFileEntry   `json:"file,omitempty"`
	KV     map[string]interface{} `json:"kv,omitempty"`
	Target bool                   `json:"target,omitempty"`
}

func (e Entry) Changed(oe Entry) bool {
	if e.Name != oe.Name {
		// fmt.Println("changed bc diff name")
		return true
	}

	if (e.File == nil) != (oe.File == nil) {
		// fmt.Println("changed bc diff file nil-ness")
		return true
	}
	if e.File != nil {
		sf, of := e.File, oe.File
		if (sf.Info == nil) != (of.Info == nil) {
			// fmt.Println("changed bc diff file info nil-ness")
			return true
		}
		if sf.Info != nil &&
			sf.Info.IsDir() != of.Info.IsDir() ||
			sf.Info.Mode() != of.Info.Mode() ||
			!sf.Info.ModTime().Equal(of.Info.ModTime()) {
			// fmt.Println(sf.Info.Name(), "changed bc diff file info", sf.Info.IsDir(), of.Info.IsDir())
			return true
		}
	}

	if (e.KV == nil) != (oe.KV == nil) {
		return true
	}
	if e.KV != nil {
		kv, okv := e.KV, oe.KV
		if len(kv) != len(okv) {
			return true
		}
		for k, v := range kv {
			ov := okv[k]
			if !reflect.DeepEqual(v, ov) {
				return true
			}
		}

		for k := range okv {
			if _, ok := kv[k]; !ok {
				return true
			}
		}
	}
	return false
}

func (e Entry) WithoutTimestamps() Entry {
	return Entry{
		Name:   e.Name,
		File:   e.File.WithoutTimestamps(),
		KV:     e.KV,
		Target: e.Target,
	}
}

type stateEntries []Entry

func (se stateEntries) Len() int {
	return len(se)
}

func (se stateEntries) Swap(i, j int) {
	se[i], se[j] = se[j], se[i]
}

func (se stateEntries) Less(i, j int) bool {
	return se[i].Name < se[j].Name
}
