package gitop

type gitState struct {
	LocalID      string `json:"local_id"`
	RemoteHeadID string `json:"remote_head_id,omitempty"`
	Version      string `json:"version,omitempty"`
}
