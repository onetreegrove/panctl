package app

import (
	"context"
	"encoding/json"
	"fmt"
	"path"

	"github.com/justonetree/pan-cli/internal/config"
	"github.com/justonetree/pan-cli/internal/credential"
	"github.com/justonetree/pan-cli/pkg/contract"
	client115 "github.com/justonetree/pan-cli/providers/115/client"
	model115 "github.com/justonetree/pan-cli/providers/115/model"
	clientAliyun "github.com/justonetree/pan-cli/providers/aliyun/client"
	modelAliyun "github.com/justonetree/pan-cli/providers/aliyun/model"
	clientBaidu "github.com/justonetree/pan-cli/providers/baidu/client"
	modelBaidu "github.com/justonetree/pan-cli/providers/baidu/model"
)

type listResult struct {
	Items    []contract.FileInfo
	Total    int
	HasMore  bool
	NextPage int
}

type providerOps interface {
	MapError(err error) contract.Error
	List(ctx context.Context, dir contract.FileInfo, page, limit int) (listResult, error)
	ListChildren(ctx context.Context, dir contract.FileInfo) ([]contract.FileInfo, error)
	DownloadURL(ctx context.Context, file contract.FileInfo, userAgent string) (string, map[string][]string, error)
	Mkdir(ctx context.Context, parent contract.FileInfo, name string) (contract.FileInfo, error)
	Move(ctx context.Context, dest contract.FileInfo, files ...contract.FileInfo) error
	Copy(ctx context.Context, dest contract.FileInfo, files ...contract.FileInfo) error
	Rename(ctx context.Context, file contract.FileInfo, newName string) error
	Delete(ctx context.Context, files ...contract.FileInfo) error
	Upload(ctx context.Context, localPath string, dest contract.FileInfo, progress func(float64)) (contract.FileInfo, error)
}

type opsLister struct {
	ops          providerOps
	providerName string
}

func (l opsLister) Provider() string {
	return l.providerName
}

func (l opsLister) List(ctx context.Context, dir contract.FileInfo) ([]contract.FileInfo, error) {
	return l.ops.ListChildren(ctx, dir)
}

func getProviderOps(rt *Runtime, ctx context.Context) (providerOps, contract.Meta, error) {
	base := rt.ConfigDir
	if base == "" {
		base = config.DefaultBaseDir()
	}
	providerName := rt.providerName()
	meta := contract.Meta{Provider: providerName, Profile: rt.Profile, RequestID: requestID()}
	data, err := credential.NewFileStore(base).Load(providerName, rt.Profile)
	if err != nil {
		return nil, meta, err
	}
	switch providerName {
	case "115":
		var cred model115.Credential
		if err := json.Unmarshal(data, &cred); err != nil {
			return nil, meta, err
		}
		c := client115.New(2)
		if err := c.LoginCookie(ctx, cred); err != nil {
			return nil, meta, err
		}
		return ops115{c: c}, meta, nil
	case "baidu":
		var cred modelBaidu.Credential
		if err := json.Unmarshal(data, &cred); err != nil {
			return nil, meta, err
		}
		c := clientBaidu.New(2)
		c.ImportCredential(cred)
		return opsBaidu{c: c}, meta, nil
	case "aliyun":
		var cred modelAliyun.Credential
		if err := json.Unmarshal(data, &cred); err != nil {
			return nil, meta, err
		}
		c := clientAliyun.New(clientAliyun.Options{})
		c.ImportCredential(cred)
		return opsAliyun{c: c}, meta, nil
	default:
		return nil, meta, fmt.Errorf("unsupported provider: %s", providerName)
	}
}

type ops115 struct {
	c *client115.Client
}

func (o ops115) MapError(err error) contract.Error {
	return client115.MapError(err)
}

func (o ops115) List(ctx context.Context, dir contract.FileInfo, page, limit int) (listResult, error) {
	res, err := o.c.List(ctx, dir.ID, page, limit)
	if err != nil {
		return listResult{}, err
	}
	items := make([]contract.FileInfo, 0, len(res.Items))
	basePath := dir.Path
	for _, item := range res.Items {
		items = append(items, item.ToContract(path.Join(basePath, item.Name)))
	}
	return listResult{Items: items, Total: res.Total, HasMore: res.HasMore, NextPage: res.NextPage}, nil
}

func (o ops115) ListChildren(ctx context.Context, dir contract.FileInfo) ([]contract.FileInfo, error) {
	var all []contract.FileInfo
	page := 1
	for {
		res, err := o.List(ctx, dir, page, 1150)
		if err != nil {
			return nil, err
		}
		all = append(all, res.Items...)
		if !res.HasMore {
			break
		}
		page++
	}
	return all, nil
}

func (o ops115) DownloadURL(ctx context.Context, file contract.FileInfo, userAgent string) (string, map[string][]string, error) {
	return o.c.DownloadURL(ctx, file.PickCode, userAgent)
}

func (o ops115) Mkdir(ctx context.Context, parent contract.FileInfo, name string) (contract.FileInfo, error) {
	id, err := o.c.Mkdir(ctx, parent.ID, name)
	if err != nil {
		return contract.FileInfo{}, err
	}
	return contract.FileInfo{ID: id, Name: name, Type: contract.FileTypeDir, Path: path.Join(parent.Path, name), Provider: "115"}, nil
}

func (o ops115) Move(ctx context.Context, dest contract.FileInfo, files ...contract.FileInfo) error {
	ids := idsFromFiles(files)
	return o.c.Move(ctx, dest.ID, ids...)
}

func (o ops115) Copy(ctx context.Context, dest contract.FileInfo, files ...contract.FileInfo) error {
	ids := idsFromFiles(files)
	return o.c.Copy(ctx, dest.ID, ids...)
}

func (o ops115) Rename(ctx context.Context, file contract.FileInfo, newName string) error {
	return o.c.Rename(ctx, file.ID, newName)
}

func (o ops115) Delete(ctx context.Context, files ...contract.FileInfo) error {
	ids := idsFromFiles(files)
	return o.c.Delete(ctx, ids...)
}

func (o ops115) Upload(ctx context.Context, localPath string, dest contract.FileInfo, progress func(float64)) (contract.FileInfo, error) {
	file, err := o.c.Upload(ctx, localPath, dest.ID, progress)
	if err != nil {
		return contract.FileInfo{}, err
	}
	return file.ToContract(path.Join(dest.Path, file.Name)), nil
}

type opsBaidu struct {
	c *clientBaidu.Client
}

func (o opsBaidu) MapError(err error) contract.Error {
	return clientBaidu.MapError(err)
}

func (o opsBaidu) List(ctx context.Context, dir contract.FileInfo, page, limit int) (listResult, error) {
	dirPath := dir.Path
	if dirPath == "" || dirPath == "0" {
		dirPath = "/"
	}
	res, err := o.c.List(ctx, dirPath, page, limit)
	if err != nil {
		return listResult{}, err
	}
	items := make([]contract.FileInfo, 0, len(res.Items))
	for _, item := range res.Items {
		items = append(items, item.ToContract(item.Path))
	}
	return listResult{Items: items, Total: res.Total, HasMore: res.HasMore, NextPage: res.NextPage}, nil
}

func (o opsBaidu) ListChildren(ctx context.Context, dir contract.FileInfo) ([]contract.FileInfo, error) {
	var all []contract.FileInfo
	page := 1
	for {
		res, err := o.List(ctx, dir, page, 1000)
		if err != nil {
			return nil, err
		}
		all = append(all, res.Items...)
		if !res.HasMore {
			break
		}
		page++
	}
	return all, nil
}

func (o opsBaidu) DownloadURL(ctx context.Context, file contract.FileInfo, userAgent string) (string, map[string][]string, error) {
	return o.c.DownloadURL(ctx, file.ID, userAgent)
}

func (o opsBaidu) Mkdir(ctx context.Context, parent contract.FileInfo, name string) (contract.FileInfo, error) {
	file, err := o.c.Mkdir(ctx, parent.Path, name)
	if err != nil {
		return contract.FileInfo{}, err
	}
	return file.ToContract(file.Path), nil
}

func (o opsBaidu) Move(ctx context.Context, dest contract.FileInfo, files ...contract.FileInfo) error {
	return o.c.Move(ctx, dest.Path, baiduFiles(files)...)
}

func (o opsBaidu) Copy(ctx context.Context, dest contract.FileInfo, files ...contract.FileInfo) error {
	return o.c.Copy(ctx, dest.Path, baiduFiles(files)...)
}

func (o opsBaidu) Rename(ctx context.Context, file contract.FileInfo, newName string) error {
	return o.c.Rename(ctx, baiduFile(file), newName)
}

func (o opsBaidu) Delete(ctx context.Context, files ...contract.FileInfo) error {
	return o.c.Delete(ctx, baiduFiles(files)...)
}

func (o opsBaidu) Upload(ctx context.Context, localPath string, dest contract.FileInfo, progress func(float64)) (contract.FileInfo, error) {
	file, err := o.c.Upload(ctx, localPath, dest.Path, progress)
	if err != nil {
		return contract.FileInfo{}, err
	}
	return file.ToContract(file.Path), nil
}

func idsFromFiles(files []contract.FileInfo) []string {
	ids := make([]string, 0, len(files))
	for _, file := range files {
		ids = append(ids, file.ID)
	}
	return ids
}

func baiduFiles(files []contract.FileInfo) []modelBaidu.File {
	result := make([]modelBaidu.File, 0, len(files))
	for _, file := range files {
		result = append(result, baiduFile(file))
	}
	return result
}

func baiduFile(file contract.FileInfo) modelBaidu.File {
	return modelBaidu.File{
		ID:    file.ID,
		Name:  file.Name,
		Path:  file.Path,
		IsDir: file.Type == contract.FileTypeDir,
		Size:  file.Size,
	}
}

type opsAliyun struct {
	c *clientAliyun.Client
}

func (o opsAliyun) MapError(err error) contract.Error {
	return clientAliyun.MapError(err)
}

func (o opsAliyun) List(ctx context.Context, dir contract.FileInfo, page, limit int) (listResult, error) {
	dirID := dir.ID
	if dirID == "" || dirID == "0" {
		dirID = "root"
	}
	res, err := o.c.List(ctx, dirID, page, limit)
	if err != nil {
		return listResult{}, err
	}
	items := make([]contract.FileInfo, 0, len(res.Items))
	basePath := dir.Path
	for _, item := range res.Items {
		items = append(items, item.ToContract(path.Join(basePath, item.Name)))
	}
	return listResult{Items: items, Total: res.Total, HasMore: res.HasMore, NextPage: res.NextPage}, nil
}

func (o opsAliyun) ListChildren(ctx context.Context, dir contract.FileInfo) ([]contract.FileInfo, error) {
	var all []contract.FileInfo
	page := 1
	for {
		res, err := o.List(ctx, dir, page, 200)
		if err != nil {
			return nil, err
		}
		all = append(all, res.Items...)
		if !res.HasMore {
			break
		}
		page++
	}
	return all, nil
}

func (o opsAliyun) DownloadURL(ctx context.Context, file contract.FileInfo, userAgent string) (string, map[string][]string, error) {
	return o.c.DownloadURL(ctx, file.ID)
}

func (o opsAliyun) Mkdir(ctx context.Context, parent contract.FileInfo, name string) (contract.FileInfo, error) {
	file, err := o.c.Mkdir(ctx, parent.ID, name)
	if err != nil {
		return contract.FileInfo{}, err
	}
	return file.ToContract(path.Join(parent.Path, file.Name)), nil
}

func (o opsAliyun) Move(ctx context.Context, dest contract.FileInfo, files ...contract.FileInfo) error {
	return o.c.Move(ctx, dest.ID, idsFromFiles(files)...)
}

func (o opsAliyun) Copy(ctx context.Context, dest contract.FileInfo, files ...contract.FileInfo) error {
	return o.c.Copy(ctx, dest.ID, idsFromFiles(files)...)
}

func (o opsAliyun) Rename(ctx context.Context, file contract.FileInfo, newName string) error {
	return o.c.Rename(ctx, file.ID, newName)
}

func (o opsAliyun) Delete(ctx context.Context, files ...contract.FileInfo) error {
	return o.c.Delete(ctx, idsFromFiles(files)...)
}

func (o opsAliyun) Upload(ctx context.Context, localPath string, dest contract.FileInfo, progress func(float64)) (contract.FileInfo, error) {
	file, err := o.c.Upload(ctx, localPath, dest.ID, progress)
	if err != nil {
		return contract.FileInfo{}, err
	}
	return file.ToContract(path.Join(dest.Path, file.Name)), nil
}
