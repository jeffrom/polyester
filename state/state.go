// Package state manages polyester states.
package state

import (
	"bufio"
	"bytes"
	"encoding/json"
	"io"
	"os"

	"github.com/mitchellh/mapstructure"
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
	b, err := json.MarshalIndent(s, "", "  ")
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

func (s State) AppendKV(name string, val interface{}) (State, error) {
	res := make(map[string]interface{})
	d, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		TagName: "json",
		Result:  &res,
	})
	if err != nil {
		return s, err
	}
	if err := d.Decode(val); err != nil {
		return s, err
	}
	return s.Append(Entry{
		Name: name,
		KV:   res,
	}), nil
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

func (s State) Map(fn func(e Entry) Entry) State {
	res := make([]Entry, len(s.Entries))
	for i, e := range s.Entries {
		res[i] = fn(e)
	}
	return State{Entries: res}
}

type States struct {
	States []StatesEntry `json:"states"`
}

type StatesEntry struct {
	Op    string `json:"op,omitempty"`
	State State  `json:"state"`
}

func (ss States) Empty() bool { return len(ss.States) == 0 }

func (ss States) Append(op string, s State) States {
	ss.States = append(ss.States, StatesEntry{Op: op, State: s})
	return ss
}

func (ss States) Find(op string) []State {
	var res []State
	for _, entry := range ss.States {
		if entry.Op == op {
			res = append(res, entry.State)
		}
	}
	return res
}
