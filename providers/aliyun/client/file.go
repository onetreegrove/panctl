package client

import (
	"context"
	"fmt"
	"net/http"

	modelAliyun "github.com/justonetree/pan-cli/providers/aliyun/model"
)

type ListResult struct {
	Items    []modelAliyun.File
	Total    int
	HasMore  bool
	NextPage int
}

func (c *Client) List(ctx context.Context, parentFileID string, page, limit int) (ListResult, error) {
	if parentFileID == "" {
		parentFileID = "root"
	}
	if page < 1 {
		page = 1
	}
	if limit <= 0 {
		limit = 100
	}
	marker := ""
	var resp listResp
	for current := 1; current <= page; current++ {
		resp = listResp{}
		body := map[string]any{
			"drive_id":        c.cred.DriveID,
			"parent_file_id":  parentFileID,
			"limit":           limit,
			"marker":          marker,
			"order_by":        "updated_at",
			"order_direction": "DESC",
		}
		if _, err := c.Request(ctx, limiterList, "/adrive/v1.0/openFile/list", http.MethodPost, body, &resp); err != nil {
			return ListResult{}, err
		}
		if current < page {
			if resp.NextMarker == "" {
				return ListResult{Items: nil, Total: 0}, nil
			}
			marker = resp.NextMarker
		}
	}

	items := make([]modelAliyun.File, 0, len(resp.Items))
	for _, item := range resp.Items {
		items = append(items, modelAliyun.FromAPIResponse(item))
	}
	nextPage := 0
	if resp.NextMarker != "" {
		nextPage = page + 1
	}
	return ListResult{
		Items:    items,
		Total:    len(items),
		HasMore:  resp.NextMarker != "",
		NextPage: nextPage,
	}, nil
}

func (c *Client) DownloadURL(ctx context.Context, fileID string) (string, map[string][]string, error) {
	var resp struct {
		URL        string            `json:"url"`
		StreamsURL map[string]string `json:"streamsUrl"`
	}
	body := map[string]any{
		"drive_id":   c.cred.DriveID,
		"file_id":    fileID,
		"expire_sec": 14400,
	}
	if _, err := c.Request(ctx, limiterLink, "/adrive/v1.0/openFile/getDownloadUrl", http.MethodPost, body, &resp); err != nil {
		return "", nil, err
	}
	urlStr := resp.URL
	if urlStr == "" && resp.StreamsURL != nil {
		urlStr = resp.StreamsURL["jpeg"]
		if urlStr == "" {
			urlStr = resp.StreamsURL["mov"]
		}
	}
	if urlStr == "" {
		return "", nil, fmt.Errorf("aliyun download link not found")
	}
	return urlStr, map[string][]string{"Referer": {"https://www.alipan.com/"}}, nil
}

func (c *Client) Mkdir(ctx context.Context, parentID, name string) (modelAliyun.File, error) {
	var resp modelAliyun.APIFile
	body := map[string]any{
		"drive_id":        c.cred.DriveID,
		"parent_file_id":  parentID,
		"name":            name,
		"type":            "folder",
		"check_name_mode": "refuse",
	}
	if _, err := c.Request(ctx, limiterOther, "/adrive/v1.0/openFile/create", http.MethodPost, body, &resp); err != nil {
		return modelAliyun.File{}, err
	}
	return modelAliyun.FromAPIResponse(resp), nil
}

func (c *Client) Move(ctx context.Context, destID string, fileIDs ...string) error {
	for _, fileID := range fileIDs {
		body := map[string]any{
			"drive_id":          c.cred.DriveID,
			"file_id":           fileID,
			"to_parent_file_id": destID,
			"check_name_mode":   "ignore",
		}
		if _, err := c.Request(ctx, limiterOther, "/adrive/v1.0/openFile/move", http.MethodPost, body, nil); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) Copy(ctx context.Context, destID string, fileIDs ...string) error {
	for _, fileID := range fileIDs {
		body := map[string]any{
			"drive_id":          c.cred.DriveID,
			"file_id":           fileID,
			"to_parent_file_id": destID,
			"auto_rename":       false,
		}
		if _, err := c.Request(ctx, limiterOther, "/adrive/v1.0/openFile/copy", http.MethodPost, body, nil); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) Rename(ctx context.Context, fileID, newName string) error {
	body := map[string]any{
		"drive_id": c.cred.DriveID,
		"file_id":  fileID,
		"name":     newName,
	}
	_, err := c.Request(ctx, limiterOther, "/adrive/v1.0/openFile/update", http.MethodPost, body, nil)
	return err
}

func (c *Client) Delete(ctx context.Context, fileIDs ...string) error {
	for _, fileID := range fileIDs {
		body := map[string]any{
			"drive_id": c.cred.DriveID,
			"file_id":  fileID,
		}
		if _, err := c.Request(ctx, limiterOther, "/adrive/v1.0/openFile/recyclebin/trash", http.MethodPost, body, nil); err != nil {
			return err
		}
	}
	return nil
}

type listResp struct {
	Items      []modelAliyun.APIFile `json:"items"`
	NextMarker string                `json:"next_marker"`
}
