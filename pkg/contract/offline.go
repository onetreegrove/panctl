package contract

type OfflineStatus string

const (
	OfflinePending OfflineStatus = "pending"
	OfflineRunning OfflineStatus = "running"
	OfflineDone    OfflineStatus = "done"
	OfflineFailed  OfflineStatus = "failed"
	OfflineUnknown OfflineStatus = "unknown"
)

type OfflineTask struct {
	GID      string         `json:"gid"`
	Name     string         `json:"name,omitempty"`
	Status   OfflineStatus  `json:"status"`
	Progress float64        `json:"progress"`
	Size     int64          `json:"size,omitempty"`
	FileID   string         `json:"file_id,omitempty"`
	Raw      map[string]any `json:"raw,omitempty"`
}
