package resolver

import (
	"context"
	"testing"

	"github.com/onetreegrove/panctl/pkg/contract"
)

type fakeLister struct {
	children map[string][]contract.FileInfo
}

func (f fakeLister) Provider() string {
	return "115"
}

func (f fakeLister) List(ctx context.Context, dir contract.FileInfo) ([]contract.FileInfo, error) {
	return f.children[dir.ID], nil
}

type pathLister struct {
	children map[string][]contract.FileInfo
	seen     []string
}

func (p *pathLister) Provider() string {
	return "baidu"
}

func (p *pathLister) List(ctx context.Context, dir contract.FileInfo) ([]contract.FileInfo, error) {
	p.seen = append(p.seen, dir.Path)
	return p.children[dir.Path], nil
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
		"0": {{ID: "1", Name: "电影", Type: contract.FileTypeDir, Path: "/电影", Provider: "115"}},
		"1": {{ID: "2", Name: "demo.mp4", Type: contract.FileTypeFile, Path: "/电影/demo.mp4", Provider: "115"}},
	}}
	got, err := Resolve(context.Background(), l, "/电影/demo.mp4")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got.ID != "2" {
		t.Fatalf("id = %q", got.ID)
	}
}

func TestResolvePathPassesCurrentFileInfoToLister(t *testing.T) {
	l := &pathLister{children: map[string][]contract.FileInfo{
		"/":     {{ID: "fsid-docs", Name: "docs", Type: contract.FileTypeDir, Path: "/docs", Provider: "baidu"}},
		"/docs": {{ID: "fsid-demo", Name: "demo.txt", Type: contract.FileTypeFile, Path: "/docs/demo.txt", Provider: "baidu"}},
	}}

	got, err := Resolve(context.Background(), l, "/docs/demo.txt")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got.ID != "fsid-demo" {
		t.Fatalf("id = %q", got.ID)
	}
	if len(l.seen) != 2 || l.seen[0] != "/" || l.seen[1] != "/docs" {
		t.Fatalf("lister saw paths = %+v", l.seen)
	}
}

func TestResolveRootUsesListerProvider(t *testing.T) {
	got, err := Resolve(context.Background(), &pathLister{}, "/")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got.Provider != "baidu" {
		t.Fatalf("provider = %q", got.Provider)
	}
}
