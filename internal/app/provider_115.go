package app

import (
	"context"
	"encoding/json"

	"github.com/onetreegrove/panctl/internal/config"
	"github.com/onetreegrove/panctl/internal/credential"
	"github.com/onetreegrove/panctl/pkg/contract"
	client115 "github.com/onetreegrove/panctl/providers/115/client"
	model115 "github.com/onetreegrove/panctl/providers/115/model"
)

type resolverLister struct {
	ctx context.Context
	c   *client115.Client
}

func (rl resolverLister) Provider() string {
	return "115"
}

func (rl resolverLister) List(ctx context.Context, dir contract.FileInfo) ([]contract.FileInfo, error) {
	return rl.listByID(ctx, dir.ID)
}

func (rl resolverLister) listByID(ctx context.Context, dirID string) ([]contract.FileInfo, error) {
	var allFiles []contract.FileInfo
	page := 1
	limit := 1150
	for {
		res, err := rl.c.List(ctx, dirID, page, limit)
		if err != nil {
			return nil, err
		}
		for _, item := range res.Items {
			allFiles = append(allFiles, item.ToContract(""))
		}
		if !res.HasMore {
			break
		}
		page++
	}
	return allFiles, nil
}

func getClient(rt *Runtime, ctx context.Context) (*client115.Client, contract.Meta, error) {
	base := rt.ConfigDir
	if base == "" {
		base = config.DefaultBaseDir()
	}
	store := credential.NewFileStore(base)
	data, err := store.Load("115", rt.Profile)
	meta := contract.Meta{Provider: "115", Profile: rt.Profile, RequestID: requestID()}
	if err != nil {
		return nil, meta, err
	}
	var cred model115.Credential
	if err := json.Unmarshal(data, &cred); err != nil {
		return nil, meta, err
	}
	c := client115.New(2)
	if err := c.LoginCookie(ctx, cred); err != nil {
		return nil, meta, err
	}
	return c, meta, nil
}
