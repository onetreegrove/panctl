package model

import (
	"testing"
	"time"
)

func TestFileToContract(t *testing.T) {
	updated := time.Unix(1717200000, 0)
	got := File{
		ID:        "123",
		Name:      "demo.mp4",
		IsDir:     false,
		Size:      10,
		SHA1:      "ABC",
		PickCode:  "pick",
		ThumbURL:  "thumb",
		UpdatedAt: updated,
	}.ToContract("/电影/demo.mp4")

	if got.ID != "123" || got.Name != "demo.mp4" || got.Type != "file" || got.PickCode != "pick" {
		t.Fatalf("file info = %+v", got)
	}
}
