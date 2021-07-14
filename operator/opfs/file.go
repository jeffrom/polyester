package opfs

import (
	"encoding/json"
	"io/fs"
	"time"
)

type StateFileEntry struct {
	Info     fs.FileInfo
	SHA256   []byte
	Contents []byte
	// ZeroTime bool
}

func (f StateFileEntry) MarshalJSON() ([]byte, error) {
	inf := f.Info
	if inf == nil {
		return json.Marshal(nil)
	}
	modTime := inf.ModTime()
	// if f.ZeroTime {
	// 	modTime = time.Time{}
	// }
	sfi := StateFileInfo{
		RawName:    inf.Name(),
		RawModTime: modTime,
		RawMode:    uint32(inf.Mode()),
		RawSize:    inf.Size(),
		SHA256:     f.SHA256,
		Contents:   f.Contents,
	}
	return json.Marshal(sfi)
}

func (f *StateFileEntry) UnmarshalJSON(b []byte) error {
	sfi := &StateFileInfo{}
	if err := json.Unmarshal(b, sfi); err != nil {
		return err
	}
	next := StateFileEntry{
		Info:   sfi,
		SHA256: sfi.SHA256,
	}
	*f = next
	return nil
}

type StateFileInfo struct {
	RawName    string    `json:"name"`
	RawModTime time.Time `json:"mtime,omitempty"`
	RawMode    uint32    `json:"mode,omitempty"`
	RawSize    int64     `json:"size,omitempty"`
	SHA256     []byte    `json:"checksum,omitempty"`
	Contents   []byte    `json:"contents,omitempty"`
}

func (sfi StateFileInfo) Name() string       { return sfi.RawName }
func (sfi StateFileInfo) Size() int64        { return sfi.RawSize }
func (sfi StateFileInfo) Mode() fs.FileMode  { return fs.FileMode(sfi.RawMode) }
func (sfi StateFileInfo) IsDir() bool        { return fs.FileMode(sfi.RawMode).IsDir() }
func (sfi StateFileInfo) ModTime() time.Time { return sfi.RawModTime }
func (sfi StateFileInfo) Sys() interface{}   { return nil }
