package model

import (
	"testing"

	"github.com/onetreegrove/panctl/pkg/contract"
)

func TestFileToContract(t *testing.T) {
	file := File{
		ID:        "123",
		Name:      "demo.txt",
		Path:      "/docs/demo.txt",
		IsDir:     false,
		Size:      42,
		MD5:       "abcdef",
		ThumbURL:  "https://example.com/thumb.jpg",
		CreatedAt: 100,
		UpdatedAt: 200,
	}

	got := file.ToContract("")

	if got.Provider != "baidu" {
		t.Fatalf("provider = %q", got.Provider)
	}
	if got.ID != "123" || got.Name != "demo.txt" || got.Path != "/docs/demo.txt" {
		t.Fatalf("unexpected file fields: %+v", got)
	}
	if got.Type != contract.FileTypeFile {
		t.Fatalf("type = %q", got.Type)
	}
	if got.Raw["md5"] != "abcdef" {
		t.Fatalf("raw md5 missing: %+v", got.Raw)
	}
}

func TestDirToContract(t *testing.T) {
	dir := File{ID: "1", Name: "docs", Path: "/docs", IsDir: true}

	got := dir.ToContract("/override")

	if got.Type != contract.FileTypeDir {
		t.Fatalf("type = %q", got.Type)
	}
	if got.Path != "/override" {
		t.Fatalf("path = %q", got.Path)
	}
}
