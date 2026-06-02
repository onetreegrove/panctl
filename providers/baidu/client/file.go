package client

import (
	"context"
	"encoding/json"
	"fmt"
	"path"
	"strconv"

	modelBaidu "github.com/justonetree/pan-cli/providers/baidu/model"
)

type ListResult struct {
	Items    []modelBaidu.File
	Total    int
	HasMore  bool
	NextPage int
}

func (c *Client) List(ctx context.Context, dirPath string, page, limit int) (ListResult, error) {
	if dirPath == "" {
		dirPath = "/"
	}
	start := (page - 1) * limit
	var resp struct {
		List []modelBaidu.APIFile `json:"list"`
	}
	_, err := c.get(ctx, "/xpan/file", map[string]string{
		"method": "list",
		"dir":    dirPath,
		"web":    "web",
		"start":  strconv.Itoa(start),
		"limit":  strconv.Itoa(limit),
	}, &resp)
	if err != nil {
		return ListResult{}, err
	}
	items := make([]modelBaidu.File, 0, len(resp.List))
	for _, item := range resp.List {
		items = append(items, modelBaidu.FromAPIFile(item))
	}
	hasMore := len(resp.List) == limit
	nextPage := 0
	if hasMore {
		nextPage = page + 1
	}
	return ListResult{Items: items, Total: start + len(items), HasMore: hasMore, NextPage: nextPage}, nil
}

func (c *Client) DownloadURL(ctx context.Context, fsID, userAgent string) (string, map[string][]string, error) {
	var resp struct {
		List []struct {
			Dlink string `json:"dlink"`
		} `json:"list"`
	}
	_, err := c.get(ctx, "/xpan/multimedia", map[string]string{
		"method": "filemetas",
		"fsids":  fmt.Sprintf("[%s]", fsID),
		"dlink":  "1",
	}, &resp)
	if err != nil {
		return "", nil, err
	}
	if len(resp.List) == 0 || resp.List[0].Dlink == "" {
		return "", nil, fmt.Errorf("baidu download link not found")
	}
	urlStr := resp.List[0].Dlink + "&access_token=" + c.cred.AccessToken
	headResp, err := c.http.R().
		SetHeader("User-Agent", "pan.baidu.com").
		SetDoNotParseResponse(true).
		Head(urlStr)
	if err == nil && headResp.Header().Get("location") != "" {
		urlStr = headResp.Header().Get("location")
	}
	return urlStr, map[string][]string{"User-Agent": {userAgent}}, nil
}

func (c *Client) Mkdir(ctx context.Context, parentPath, name string) (modelBaidu.File, error) {
	var file modelBaidu.APIFile
	_, err := c.create(ctx, path.Join(parentPath, name), 0, true, "", "", &file)
	return modelBaidu.FromAPIFile(file), err
}

func (c *Client) Move(ctx context.Context, destPath string, files ...modelBaidu.File) error {
	payload := make([]map[string]string, 0, len(files))
	for _, file := range files {
		payload = append(payload, map[string]string{"path": file.Path, "dest": destPath, "newname": file.Name})
	}
	return c.manage(ctx, "move", payload)
}

func (c *Client) Copy(ctx context.Context, destPath string, files ...modelBaidu.File) error {
	payload := make([]map[string]string, 0, len(files))
	for _, file := range files {
		payload = append(payload, map[string]string{"path": file.Path, "dest": destPath, "newname": file.Name})
	}
	return c.manage(ctx, "copy", payload)
}

func (c *Client) Rename(ctx context.Context, file modelBaidu.File, newName string) error {
	return c.manage(ctx, "rename", []map[string]string{{"path": file.Path, "newname": newName}})
}

func (c *Client) Delete(ctx context.Context, files ...modelBaidu.File) error {
	paths := make([]string, 0, len(files))
	for _, file := range files {
		paths = append(paths, file.Path)
	}
	return c.manage(ctx, "delete", paths)
}

func (c *Client) manage(ctx context.Context, opera string, filelist any) error {
	payload, err := jsonMarshalString(filelist)
	if err != nil {
		return err
	}
	_, err = c.postForm(ctx, "/xpan/file", map[string]string{
		"method": "filemanager",
		"opera":  opera,
	}, map[string]string{
		"async":    "0",
		"filelist": payload,
		"ondup":    "fail",
	}, nil)
	return err
}

func (c *Client) create(ctx context.Context, filePath string, size int64, isDir bool, uploadID, blockList string, result any) ([]byte, error) {
	isDirValue := "0"
	if isDir {
		isDirValue = "1"
	}
	form := map[string]string{
		"path":  filePath,
		"size":  strconv.FormatInt(size, 10),
		"isdir": isDirValue,
		"rtype": "3",
	}
	if uploadID != "" {
		form["uploadid"] = uploadID
	}
	if blockList != "" {
		form["block_list"] = blockList
	}
	return c.postForm(ctx, "/xpan/file", map[string]string{"method": "create"}, form, result)
}

func jsonMarshalString(v any) (string, error) {
	b, err := json.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(b), nil
}
