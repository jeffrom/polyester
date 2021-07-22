package state

import (
	"bytes"
	"reflect"

	"github.com/jeffrom/polyester/operator/opfs"
)

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
		if !bytes.Equal(sf.SHA256, of.SHA256) {
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
