package client

import (
	"context"
	"fmt"
	"io"
	"math"
	"net/http"
	"os"
	"time"

	modelAliyun "github.com/onetreegrove/panctl/providers/aliyun/model"
)

type uploadCreateResp struct {
	FileID       string     `json:"file_id"`
	UploadID     string     `json:"upload_id"`
	RapidUpload  bool       `json:"rapid_upload"`
	PartInfoList []partInfo `json:"part_info_list"`
}

type partInfo struct {
	PartNumber int    `json:"part_number"`
	UploadURL  string `json:"upload_url"`
}

func (c *Client) Upload(ctx context.Context, localPath, destDirID string, progress func(float64)) (modelAliyun.File, error) {
	f, err := os.Open(localPath)
	if err != nil {
		return modelAliyun.File{}, err
	}
	defer f.Close()
	stat, err := f.Stat()
	if err != nil {
		return modelAliyun.File{}, err
	}
	partSize := PartSize(stat.Size())
	partCount := int(math.Ceil(float64(stat.Size()) / float64(partSize)))
	if partCount == 0 {
		partCount = 1
	}
	partInfos := make([]map[string]int, 0, partCount)
	for i := 1; i <= partCount; i++ {
		partInfos = append(partInfos, map[string]int{"part_number": i})
	}
	createBody := map[string]any{
		"drive_id":        c.cred.DriveID,
		"parent_file_id":  destDirID,
		"name":            stat.Name(),
		"type":            "file",
		"check_name_mode": "ignore",
		"size":            stat.Size(),
		"part_info_list":  partInfos,
	}
	if stat.Size() > 100*kb {
		preHash, err := PreHash(ctx, f)
		if err != nil {
			return modelAliyun.File{}, err
		}
		createBody["pre_hash"] = preHash
		if _, err := f.Seek(0, io.SeekStart); err != nil {
			return modelAliyun.File{}, err
		}
	}

	createResp, err := c.createUpload(ctx, createBody)
	if err != nil {
		if apiErr, ok := err.(errResp); !ok || apiErr.Code != "PreHashMatched" {
			return modelAliyun.File{}, err
		}
		if err := c.fillRapidUploadProof(ctx, f, stat.Size(), createBody); err != nil {
			return modelAliyun.File{}, err
		}
		createResp, err = c.createUpload(ctx, createBody)
		if err != nil {
			return modelAliyun.File{}, err
		}
	}

	if !createResp.RapidUpload {
		if err := c.uploadParts(ctx, f, partSize, createResp.PartInfoList, progress); err != nil {
			return modelAliyun.File{}, err
		}
	} else if progress != nil {
		progress(100)
	}
	return c.completeUpload(ctx, createResp.FileID, createResp.UploadID)
}

func (c *Client) createUpload(ctx context.Context, body map[string]any) (uploadCreateResp, error) {
	var resp uploadCreateResp
	_, err := c.Request(ctx, limiterOther, "/adrive/v1.0/openFile/create", http.MethodPost, body, &resp)
	return resp, err
}

func (c *Client) fillRapidUploadProof(ctx context.Context, f *os.File, size int64, body map[string]any) error {
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return err
	}
	fullHash, err := SHA1Hex(ctx, f)
	if err != nil {
		return err
	}
	proof, err := ProofCode(c.cred.AccessToken, size, f)
	if err != nil {
		return err
	}
	delete(body, "pre_hash")
	body["content_hash"] = fullHash
	body["content_hash_name"] = "sha1"
	body["proof_version"] = "v1"
	body["proof_code"] = proof
	_, err = f.Seek(0, io.SeekStart)
	return err
}

func (c *Client) uploadParts(ctx context.Context, f *os.File, partSize int64, parts []partInfo, progress func(float64)) error {
	started := time.Now()
	for i, part := range parts {
		if err := ctx.Err(); err != nil {
			return err
		}
		if part.UploadURL == "" {
			return fmt.Errorf("aliyun upload url missing for part %d", part.PartNumber)
		}
		if time.Since(started) > 50*time.Minute {
			started = time.Now()
		}
		offset := int64(i) * partSize
		size := partSize
		stat, err := f.Stat()
		if err != nil {
			return err
		}
		if offset+size > stat.Size() {
			size = stat.Size() - offset
		}
		req, err := http.NewRequestWithContext(ctx, http.MethodPut, part.UploadURL, io.NewSectionReader(f, offset, size))
		if err != nil {
			return err
		}
		res, err := c.http.GetClient().Do(req)
		if err != nil {
			return err
		}
		_ = res.Body.Close()
		if res.StatusCode != http.StatusOK && res.StatusCode != http.StatusConflict {
			return fmt.Errorf("aliyun upload part status %d", res.StatusCode)
		}
		if progress != nil {
			progress(float64(i+1) * 100 / float64(len(parts)))
		}
	}
	return nil
}

func (c *Client) completeUpload(ctx context.Context, fileID, uploadID string) (modelAliyun.File, error) {
	var resp modelAliyun.APIFile
	body := map[string]any{
		"drive_id":  c.cred.DriveID,
		"file_id":   fileID,
		"upload_id": uploadID,
	}
	if _, err := c.Request(ctx, limiterOther, "/adrive/v1.0/openFile/complete", http.MethodPost, body, &resp); err != nil {
		return modelAliyun.File{}, err
	}
	return modelAliyun.FromAPIResponse(resp), nil
}
