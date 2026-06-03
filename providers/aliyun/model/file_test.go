package model

import (
	"testing"
	"time"

	"github.com/onetreegrove/panctl/pkg/contract"
)

func TestFileToContract(t *testing.T) {
	created := time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC)
	updated := time.Date(2026, 6, 2, 11, 0, 0, 0, time.UTC)
	file := File{
		DriveID:      "drive_1",
		FileID:       "file_1",
		ParentFileID: "root",
		Name:         "demo.mp4",
		Size:         1024,
		Type:         "file",
		ContentHash:  "abc123",
		Thumbnail:    "https://thumb.example/demo.jpg",
		Category:     "video",
		CreatedAt:    created,
		UpdatedAt:    updated,
	}
	got := FromAPIFile(file).ToContract("/备份/demo.mp4")
	if got.ID != "file_1" || got.Name != "demo.mp4" || got.Type != contract.FileTypeFile || got.Provider != "aliyun" {
		t.Fatalf("unexpected contract file: %+v", got)
	}
	if got.Raw["drive_id"] != "drive_1" || got.Raw["content_hash"] != "abc123" {
		t.Fatalf("expected raw metadata, got %+v", got.Raw)
	}
}

func TestFolderToContract(t *testing.T) {
	got := FromAPIFile(File{FileID: "dir_1", Name: "资料", Type: "folder"}).ToContract("/资料")
	if got.Type != contract.FileTypeDir {
		t.Fatalf("expected dir, got %s", got.Type)
	}
}
