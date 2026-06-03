package resolver

import (
	"context"
	"errors"
	"path"
	"regexp"
	"strings"

	"github.com/onetreegrove/panctl/pkg/contract"
)

type Lister interface {
	Provider() string
	List(ctx context.Context, dir contract.FileInfo) ([]contract.FileInfo, error)
}

var numericID = regexp.MustCompile(`^\d+$`)

func Resolve(ctx context.Context, lister Lister, target string) (contract.FileInfo, error) {
	providerName := lister.Provider()
	if target == "" || target == "/" {
		return contract.FileInfo{ID: "0", Name: "/", Type: contract.FileTypeDir, Path: "/", Provider: providerName}, nil
	}
	if numericID.MatchString(target) {
		return contract.FileInfo{ID: target, Provider: providerName}, nil
	}
	if !strings.HasPrefix(target, "/") {
		return contract.FileInfo{}, errors.New("target must be an id or absolute path")
	}
	current := contract.FileInfo{ID: "0", Name: "/", Type: contract.FileTypeDir, Path: "/", Provider: providerName}
	for _, part := range strings.Split(strings.Trim(path.Clean(target), "/"), "/") {
		children, err := lister.List(ctx, current)
		if err != nil {
			return contract.FileInfo{}, err
		}
		found := false
		for _, child := range children {
			if child.Name == part {
				current = child
				found = true
				break
			}
		}
		if !found {
			return contract.FileInfo{}, errors.New("not found")
		}
	}
	return current, nil
}
