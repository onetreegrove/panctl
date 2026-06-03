package model

import (
	"path"
	"time"

	"github.com/justonetree/pan-cli/pkg/contract"
)

type File struct {
	ID           string
	DriveID      string
	FileID       string
	ParentFileID string
	Name         string
	Size         int64
	Type         string
	ContentHash  string
	Thumbnail    string
	Category     string
	CreatedAt    time.Time
	UpdatedAt    time.Time
}

func (f File) ToContract(resolvedPath string) contract.FileInfo {
	if resolvedPath == "" && f.Name != "" {
		resolvedPath = "/" + f.Name
	}
	fileType := contract.FileTypeFile
	if f.Type == "folder" {
		fileType = contract.FileTypeDir
	}
	var createdAt *time.Time
	if !f.CreatedAt.IsZero() {
		t := f.CreatedAt
		createdAt = &t
	}
	var updatedAt *time.Time
	if !f.UpdatedAt.IsZero() {
		t := f.UpdatedAt
		updatedAt = &t
	}
	raw := map[string]any{}
	if f.DriveID != "" {
		raw["drive_id"] = f.DriveID
	}
	if f.ParentFileID != "" {
		raw["parent_file_id"] = f.ParentFileID
	}
	if f.Category != "" {
		raw["category"] = f.Category
	}
	if f.ContentHash != "" {
		raw["content_hash"] = f.ContentHash
	}
	return contract.FileInfo{
		ID:        f.ID,
		Name:      f.Name,
		Type:      fileType,
		Path:      path.Clean(resolvedPath),
		Size:      f.Size,
		SHA1:      f.ContentHash,
		ThumbURL:  f.Thumbnail,
		CreatedAt: createdAt,
		UpdatedAt: updatedAt,
		Provider:  "aliyun",
		Raw:       raw,
	}
}

type APIFile struct {
	DriveID      string    `json:"drive_id"`
	FileID       string    `json:"file_id"`
	ParentFileID string    `json:"parent_file_id"`
	Name         string    `json:"name"`
	FileName     string    `json:"file_name"`
	Size         int64     `json:"size"`
	Type         string    `json:"type"`
	ContentHash  string    `json:"content_hash"`
	Thumbnail    string    `json:"thumbnail"`
	Category     string    `json:"category"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

func FromAPIFile(src File) File {
	if src.ID == "" {
		src.ID = src.FileID
	}
	if src.Name == "" {
		src.Name = path.Base(src.FileID)
	}
	return src
}

func FromAPIResponse(src APIFile) File {
	name := src.Name
	if name == "" {
		name = src.FileName
	}
	return File{
		ID:           src.FileID,
		DriveID:      src.DriveID,
		FileID:       src.FileID,
		ParentFileID: src.ParentFileID,
		Name:         name,
		Size:         src.Size,
		Type:         src.Type,
		ContentHash:  src.ContentHash,
		Thumbnail:    src.Thumbnail,
		Category:     src.Category,
		CreatedAt:    src.CreatedAt,
		UpdatedAt:    src.UpdatedAt,
	}
}
