package client

import (
	"context"
	"strconv"

	driver115 "github.com/SheltonZhu/115driver/pkg/driver"
	model115 "github.com/justonetree/pan-cli/providers/115/model"
)

type ListResult struct {
	Items    []model115.File
	Total    int
	HasMore  bool
	NextPage int
}

type fileInfoWithThumb struct {
	driver115.FileInfo
	ThumbURL string `json:"u"`
}

type fileListRespWithThumb struct {
	driver115.BasicResp
	CategoryID driver115.IntString `json:"cid"`
	Count      int                 `json:"count"`
	Offset     int                 `json:"offset"`
	Files      []fileInfoWithThumb `json:"data"`
}

func (c *Client) List(ctx context.Context, dirID string, page, limit int) (ListResult, error) {
	if err := c.init(ctx); err != nil {
		return ListResult{}, err
	}
	return listPage(ctx, c.raw, dirID, page, limit)
}

func (c *Client) DownloadURL(ctx context.Context, pickCode, userAgent string) (string, map[string][]string, error) {
	if err := c.init(ctx); err != nil {
		return "", nil, err
	}
	info, err := c.raw.DownloadWithUA(pickCode, userAgent)
	if err != nil {
		return "", nil, err
	}
	return info.Url.Url, info.Header, nil
}

func (c *Client) Mkdir(ctx context.Context, parentID, name string) (string, error) {
	if err := c.init(ctx); err != nil {
		return "", err
	}
	return c.raw.Mkdir(parentID, name)
}

func (c *Client) Move(ctx context.Context, dirID string, fileIDs ...string) error {
	if err := c.init(ctx); err != nil {
		return err
	}
	return c.raw.Move(dirID, fileIDs...)
}

func (c *Client) Copy(ctx context.Context, dirID string, fileIDs ...string) error {
	if err := c.init(ctx); err != nil {
		return err
	}
	return c.raw.Copy(dirID, fileIDs...)
}

func (c *Client) Rename(ctx context.Context, fileID, newName string) error {
	if err := c.init(ctx); err != nil {
		return err
	}
	return c.raw.Rename(fileID, newName)
}

func (c *Client) Delete(ctx context.Context, fileIDs ...string) error {
	if err := c.init(ctx); err != nil {
		return err
	}
	return c.raw.Delete(fileIDs...)
}

func listPage(ctx context.Context, raw *driver115.Pan115Client, dirID string, page, limit int) (ListResult, error) {
	if dirID == "" {
		dirID = "0"
	}
	offset := int64((page - 1) * limit)
	result := fileListRespWithThumb{}
	params := map[string]string{
		"aid":              "1",
		"cid":              dirID,
		"o":                driver115.FileOrderByTime,
		"asc":              "1",
		"offset":           strconv.FormatInt(offset, 10),
		"show_dir":         "1",
		"limit":            strconv.FormatInt(int64(limit), 10),
		"snap":             "0",
		"natsort":          "0",
		"record_open_time": "1",
		"format":           "json",
		"fc_mix":           "0",
	}

	req := raw.NewRequest().
		ForceContentType("application/json;charset=UTF-8").
		SetQueryParams(params).
		SetResult(&result).
		SetContext(ctx)

	resp, err := req.Get(driver115.ApiFileList)
	if err := driver115.CheckErr(err, &result, resp); err != nil {
		return ListResult{}, err
	}

	if dirID != string(result.CategoryID) {
		return ListResult{}, driver115.ErrUnexpected
	}

	var files []model115.File
	for _, fi := range result.Files {
		f := &driver115.File{}
		f.From(&fi.FileInfo)
		thumb := fi.ThumbURL
		if thumb == "" {
			thumb = fi.FileInfo.ThumbURL
		}
		files = append(files, model115.File{
			ID:        f.FileID,
			Name:      f.Name,
			IsDir:     f.IsDirectory,
			Size:      f.Size,
			SHA1:      f.Sha1,
			PickCode:  f.PickCode,
			ThumbURL:  thumb,
			CreatedAt: f.CreateTime,
			UpdatedAt: f.UpdateTime,
		})
	}

	hasMore := offset+int64(len(result.Files)) < int64(result.Count)
	nextPage := 0
	if hasMore {
		nextPage = page + 1
	}

	return ListResult{
		Items:    files,
		Total:    result.Count,
		HasMore:  hasMore,
		NextPage: nextPage,
	}, nil
}
