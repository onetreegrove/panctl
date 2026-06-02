package resolver

import (
	"context"
	"testing"

	"github.com/justonetree/pan-cli/pkg/contract"
)

type fakeLister struct {
	children map[string][]contract.FileInfo
}

func (f fakeLister) List(ctx context.Context, dirID string) ([]contract.FileInfo, error) {
	return f.children[dirID], nil
}

func TestResolveID(t *testing.T) {
	got, err := Resolve(context.Background(), fakeLister{}, "123")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got.ID != "123" {
		t.Fatalf("id = %q", got.ID)
	}
}

func TestResolvePath(t *testing.T) {
	l := fakeLister{children: map[string][]contract.FileInfo{
		"0":   {{ID: "1", Name: "电影", Type: contract.FileTypeDir}},
		"1":   {{ID: "2", Name: "demo.mp4", Type: contract.FileTypeFile}},
	}}
	got, err := Resolve(context.Background(), l, "/电影/demo.mp4")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got.ID != "2" {
		t.Fatalf("id = %q", got.ID)
	}
}
