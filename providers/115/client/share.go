package client

import (
	"context"
	"fmt"
	"net/url"
	"strconv"
	"strings"
	"time"

	driver115 "github.com/SheltonZhu/115driver/pkg/driver"
	"github.com/onetreegrove/panctl/pkg/contract"
)

// ParseShareURL parses the share URL and extracts the share code and access code.
func ParseShareURL(shareURL string) (string, string, error) {
	u, err := url.Parse(shareURL)
	if err != nil {
		return "", "", err
	}
	parts := strings.Split(u.Path, "/")
	var shareCode string
	for i, part := range parts {
		if part == "s" && i+1 < len(parts) {
			shareCode = parts[i+1]
			break
		}
	}
	if shareCode == "" && len(parts) > 0 {
		shareCode = parts[len(parts)-1]
	}
	return shareCode, u.Query().Get("password"), nil
}

// ShareList lists files in a shared folder directory.
func (c *Client) ShareList(ctx context.Context, shareCode, receiveCode, dirID string, page, limit int) ([]contract.FileInfo, int, error) {
	if err := c.init(ctx); err != nil {
		return nil, 0, err
	}
	if dirID == "" {
		dirID = "0"
	}
	offset := (page - 1) * limit

	type shareFile struct {
		FileID     string                `json:"fid"`
		CategoryID driver115.IntString   `json:"cid"`
		FileName   string                `json:"n"`
		Type       string                `json:"ico"`
		Sha1       string                `json:"sha"`
		Size       driver115.StringInt64 `json:"s"`
		UpdateTime string                `json:"t"`
		IsFile     int                   `json:"fc"`
		ThumbURL   string                `json:"u"`
	}

	type shareSnapResp struct {
		driver115.BasicResp
		Data struct {
			Count int         `json:"count"`
			List  []shareFile `json:"list"`
		} `json:"data"`
	}

	result := shareSnapResp{}
	query := map[string]string{
		"share_code":   shareCode,
		"receive_code": receiveCode,
		"cid":          dirID,
		"limit":        strconv.Itoa(limit),
		"asc":          "0",
		"offset":       strconv.Itoa(offset),
		"format":       "json",
	}

	req := c.raw.NewRequest().
		SetQueryParams(query).
		SetHeader("referer", fmt.Sprintf("https://115cdn.com/s/%s?password=%s&", shareCode, receiveCode)).
		ForceContentType("application/json;charset=UTF-8").
		SetResult(&result).
		SetContext(ctx)

	resp, err := req.Get(driver115.ApiShareSnap)
	if err := driver115.CheckErr(err, &result, resp); err != nil {
		return nil, 0, err
	}

	var files []contract.FileInfo
	for _, sf := range result.Data.List {
		timeInt, _ := strconv.ParseInt(sf.UpdateTime, 10, 64)
		utm := time.Unix(timeInt, 0)
		isDir := (sf.IsFile == 0)
		fileID := sf.FileID
		if isDir {
			fileID = string(sf.CategoryID)
		}

		fileType := contract.FileTypeFile
		if isDir {
			fileType = contract.FileTypeDir
		}
		files = append(files, contract.FileInfo{
			ID:        fileID,
			Name:      sf.FileName,
			Type:      fileType,
			Size:      int64(sf.Size),
			SHA1:      sf.Sha1,
			ThumbURL:  sf.ThumbURL,
			UpdatedAt: &utm,
			Provider:  "115",
		})
	}

	return files, result.Data.Count, nil
}

// ShareDownloadURL gets direct download URL for a file inside a shared link.
func (c *Client) ShareDownloadURL(ctx context.Context, shareCode, receiveCode, fileID string) (string, error) {
	if err := c.init(ctx); err != nil {
		return "", err
	}

	type downloadShareResp struct {
		driver115.BasicResp
		Data driver115.SharedDownloadInfo `json:"data"`
	}
	result := downloadShareResp{}
	params := map[string]string{
		"share_code":   shareCode,
		"receive_code": receiveCode,
		"file_id":      fileID,
		"dl":           "1",
	}

	req := c.raw.NewRequest().
		SetQueryParams(params).
		ForceContentType("application/json").
		SetHeader("referer", fmt.Sprintf("https://115cdn.com/s/%s?password=%s&", shareCode, receiveCode)).
		SetResult(&result).
		SetContext(ctx)

	resp, err := req.Get(driver115.ApiDownloadGetShareUrl)
	if err := driver115.CheckErr(err, &result, resp); err != nil {
		return "", err
	}

	return result.Data.URL.URL, nil
}
