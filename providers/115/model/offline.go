package model

import "github.com/justonetree/pan-cli/pkg/contract"

type OfflineTask struct {
	GID        string
	Name       string
	StatusText string
	Progress   float64
	Size       int64
	FileID     string
}

func (t OfflineTask) ToContract() contract.OfflineTask {
	status := contract.OfflineUnknown
	switch t.StatusText {
	case "pending":
		status = contract.OfflinePending
	case "running":
		status = contract.OfflineRunning
	case "done", "success", "completed":
		status = contract.OfflineDone
	case "failed", "error":
		status = contract.OfflineFailed
	}
	return contract.OfflineTask{
		GID:      t.GID,
		Name:     t.Name,
		Status:   status,
		Progress: t.Progress,
		Size:     t.Size,
		FileID:   t.FileID,
	}
}
