package model

import (
	"time"

	"github.com/justonetree/pan-cli/pkg/contract"
)

type File struct {
	ID        string
	Name      string
	IsDir     bool
	Size      int64
	SHA1      string
	PickCode  string
	ThumbURL  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (f File) ToContract(path string) contract.FileInfo {
	fileType := contract.FileTypeFile
	if f.IsDir {
		fileType = contract.FileTypeDir
	}
	var created *time.Time
	if !f.CreatedAt.IsZero() {
		created = &f.CreatedAt
	}
	var updated *time.Time
	if !f.UpdatedAt.IsZero() {
		updated = &f.UpdatedAt
	}
	return contract.FileInfo{
		ID:        f.ID,
		Name:      f.Name,
		Type:      fileType,
		Path:      path,
		Size:      f.Size,
		SHA1:      f.SHA1,
		PickCode:  f.PickCode,
		ThumbURL:  f.ThumbURL,
		CreatedAt: created,
		UpdatedAt: updated,
		Provider:  "115",
	}
}
