package client

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path"
	"strconv"

	modelBaidu "github.com/onetreegrove/panctl/providers/baidu/model"
)

const defaultSliceSize int64 = 4 * 1024 * 1024

func (c *Client) RapidUpload(ctx context.Context, localPath, targetDirPath string) (*modelBaidu.File, error) {
	f, err := os.Open(localPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if stat.Size() < 1 {
		return nil, fmt.Errorf("empty files are not allowed by baidu netdisk")
	}
	contentMD5, err := MD5Hex(ctx, f)
	if err != nil {
		return nil, err
	}
	blockList, err := json.Marshal([]string{contentMD5})
	if err != nil {
		return nil, err
	}
	var raw modelBaidu.APIFile
	_, err = c.create(ctx, path.Join(targetDirPath, stat.Name()), stat.Size(), false, "", string(blockList), &raw)
	if err != nil {
		return nil, err
	}
	file := modelBaidu.FromAPIFile(raw)
	return &file, nil
}

func (c *Client) Upload(ctx context.Context, localPath, targetDirPath string, progress func(percent float64)) (*modelBaidu.File, error) {
	f, err := os.Open(localPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}
	if stat.Size() < 1 {
		return nil, fmt.Errorf("empty files are not allowed by baidu netdisk")
	}
	if file, err := c.RapidUpload(ctx, localPath, targetDirPath); err == nil {
		if progress != nil {
			progress(100)
		}
		return file, nil
	}
	contentMD5, err := MD5Hex(ctx, f)
	if err != nil {
		return nil, err
	}
	blockMD5s, err := SliceMD5s(ctx, f, defaultSliceSize)
	if err != nil {
		return nil, err
	}
	blockList, err := json.Marshal(blockMD5s)
	if err != nil {
		return nil, err
	}
	filePath := path.Join(targetDirPath, stat.Name())
	pre, err := c.precreate(ctx, filePath, stat.Size(), string(blockList), contentMD5)
	if err != nil {
		return nil, err
	}
	if pre.ReturnType == 2 {
		file := modelBaidu.FromAPIFile(pre.Info)
		if progress != nil {
			progress(100)
		}
		return &file, nil
	}
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}
	totalParts := len(pre.BlockList)
	for i, partSeq := range pre.BlockList {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		offset := int64(partSeq) * defaultSliceSize
		size := defaultSliceSize
		if offset+size > stat.Size() {
			size = stat.Size() - offset
		}
		if err := c.uploadSlice(ctx, filePath, pre.UploadID, partSeq, stat.Name(), io.NewSectionReader(f, offset, size)); err != nil {
			return nil, err
		}
		if progress != nil && totalParts > 0 {
			progress(float64(i+1) * 100 / float64(totalParts))
		}
	}
	var raw modelBaidu.APIFile
	_, err = c.create(ctx, filePath, stat.Size(), false, pre.UploadID, string(blockList), &raw)
	if err != nil {
		return nil, err
	}
	file := modelBaidu.FromAPIFile(raw)
	return &file, nil
}

type precreateResp struct {
	ReturnType int                `json:"return_type"`
	UploadID   string             `json:"uploadid"`
	BlockList  []int              `json:"block_list"`
	Info       modelBaidu.APIFile `json:"info"`
}

func (c *Client) precreate(ctx context.Context, filePath string, size int64, blockList, contentMD5 string) (*precreateResp, error) {
	form := map[string]string{
		"path":        filePath,
		"size":        strconv.FormatInt(size, 10),
		"isdir":       "0",
		"autoinit":    "1",
		"rtype":       "3",
		"block_list":  blockList,
		"content-md5": contentMD5,
	}
	var resp precreateResp
	_, err := c.postForm(ctx, "/xpan/file", map[string]string{"method": "precreate"}, form, &resp)
	return &resp, err
}

func (c *Client) uploadSlice(ctx context.Context, filePath, uploadID string, partSeq int, fileName string, r io.Reader) error {
	if err := c.ensureAccessToken(ctx); err != nil {
		return err
	}
	resp, err := c.http.R().
		SetContext(ctx).
		SetQueryParams(map[string]string{
			"method":       "upload",
			"access_token": c.cred.AccessToken,
			"type":         "tmpfile",
			"path":         filePath,
			"uploadid":     uploadID,
			"partseq":      strconv.Itoa(partSeq),
		}).
		SetFileReader("file", fileName, r).
		Post(defaultUploadURL + "/rest/2.0/pcs/superfile2")
	if err != nil {
		return err
	}
	if resp.StatusCode() >= 400 || jsonInt(resp.Body(), "error_code") != 0 || jsonInt(resp.Body(), "errno") != 0 {
		return fmt.Errorf("baidu upload slice failed: %s", resp.String())
	}
	return nil
}
