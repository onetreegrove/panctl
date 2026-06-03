package client

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync/atomic"
	"time"

	driver115 "github.com/SheltonZhu/115driver/pkg/driver"
	"github.com/aliyun/aliyun-oss-go-sdk/oss"
	model115 "github.com/onetreegrove/panctl/providers/115/model"
)

// Upload uploads a local file to the specified target directory ID on 115.
func (c *Client) Upload(ctx context.Context, localPath string, targetDirID string, progress func(percent float64)) (*model115.File, error) {
	if err := c.init(ctx); err != nil {
		return nil, err
	}

	f, err := os.Open(localPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	stat, err := f.Stat()
	if err != nil {
		return nil, err
	}
	fileSize := stat.Size()
	fileName := stat.Name()

	// 1. Calculate hashes
	preHash, err := PreHash(ctx, f)
	if err != nil {
		return nil, err
	}

	fullHash, err := FullHash(ctx, f, nil)
	if err != nil {
		return nil, err
	}

	// Reset read pointer
	if _, err := f.Seek(0, io.SeekStart); err != nil {
		return nil, err
	}

	// 2. Perform RapidUpload check
	initResp, err := c.raw.RapidUpload(fileSize, fileName, targetDirID, preHash, fullHash, f)
	if err != nil {
		return nil, err
	}

	// If rapid upload succeeded
	if initResp.Status == 1 {
		fileIDStr := fmt.Sprintf("%d", initResp.FileID)
		rawFile, err := c.raw.GetFile(fileIDStr)
		if err == nil && rawFile != nil {
			contractFile := convertFile(rawFile)
			return &contractFile, nil
		}
		// Fallback if GetFile fails
		return &model115.File{
			ID:       fileIDStr,
			Name:     fileName,
			IsDir:    false,
			Size:     fileSize,
			SHA1:     fullHash,
			PickCode: initResp.PickCode,
		}, nil
	}

	// 3. Perform actual OSS upload if rapid upload failed
	var fileID string
	if fileSize <= 10*1024*1024 { // <= 10 MiB
		res, err := c.uploadByOSS(ctx, &initResp.UploadOSSParams, fileSize, f, progress)
		if err != nil {
			return nil, err
		}
		fileID = res.Data.FileID
	} else { // > 10 MiB
		res, err := c.uploadByMultipart(ctx, &initResp.UploadOSSParams, fileSize, f, progress)
		if err != nil {
			return nil, err
		}
		fileID = res.Data.FileID
	}

	// 4. Retrieve the uploaded file metadata
	rawFile, err := c.raw.GetFile(fileID)
	if err != nil {
		return nil, err
	}
	contractFile := convertFile(rawFile)
	return &contractFile, nil
}

func (c *Client) uploadByOSS(ctx context.Context, params *driver115.UploadOSSParams, fileSize int64, r io.Reader, progress func(percent float64)) (*driver115.UploadResult, error) {
	ossToken, err := c.raw.GetOSSToken()
	if err != nil {
		return nil, err
	}
	ossClient, err := oss.New(c.raw.GetOSSEndpoint(c.raw.UseInternalUpload), ossToken.AccessKeyID, ossToken.AccessKeySecret)
	if err != nil {
		return nil, err
	}
	bucket, err := ossClient.Bucket(params.Bucket)
	if err != nil {
		return nil, err
	}

	var bodyBytes []byte

	// Wrap reader for context cancellation and progress updates
	var totalRead int64
	wrappedReader := &progressReader{
		r: &contextReader{ctx: ctx, r: r},
		updater: func(bytesRead int64) {
			totalRead = bytesRead
			if progress != nil && fileSize > 0 {
				progress(float64(totalRead) * 100.0 / float64(fileSize))
			}
		},
	}

	if err = bucket.PutObject(params.Object, wrappedReader, append(
		driver115.OssOption(params, ossToken),
		oss.CallbackResult(&bodyBytes),
	)...); err != nil {
		return nil, err
	}

	var uploadResult driver115.UploadResult
	if err = json.Unmarshal(bodyBytes, &uploadResult); err != nil {
		return nil, err
	}
	return &uploadResult, uploadResult.Err(string(bodyBytes))
}

func (c *Client) uploadByMultipart(ctx context.Context, params *driver115.UploadOSSParams, fileSize int64, f *os.File, progress func(percent float64)) (*driver115.UploadResult, error) {
	var (
		parts     []oss.UploadPart
		imur      oss.InitiateMultipartUploadResult
		ossClient *oss.Client
		bucket    *oss.Bucket
		ossToken  *driver115.UploadOSSTokenResp
		bodyBytes []byte
		err       error
	)

	if ossToken, err = c.raw.GetOSSToken(); err != nil {
		return nil, err
	}

	if ossClient, err = oss.New(
		c.raw.GetOSSEndpoint(c.raw.UseInternalUpload),
		ossToken.AccessKeyID,
		ossToken.AccessKeySecret,
		oss.EnableMD5(true),
		oss.EnableCRC(true),
	); err != nil {
		return nil, err
	}

	if bucket, err = ossClient.Bucket(params.Bucket); err != nil {
		return nil, err
	}

	// Split file into 10 MiB chunks
	chunkSize := int64(10 * 1024 * 1024)
	var chunks []fileChunk
	var chunkNum int = 1
	for offset := int64(0); offset < fileSize; offset += chunkSize {
		size := chunkSize
		if offset+size > fileSize {
			size = fileSize - offset
		}
		chunks = append(chunks, fileChunk{
			Number: chunkNum,
			Offset: offset,
			Size:   size,
		})
		chunkNum++
	}

	if imur, err = bucket.InitiateMultipartUpload(params.Object,
		oss.SetHeader(driver115.OssSecurityTokenHeaderName, ossToken.SecurityToken),
		oss.UserAgentHeader(driver115.OSSUserAgent),
		oss.EnableSha1(),
		oss.Sequential(),
	); err != nil {
		return nil, err
	}

	// 50 minutes token refresh ticker
	ticker := time.NewTicker(50 * time.Minute)
	defer ticker.Stop()

	completedNum := atomic.Int32{}

	// Upload chunks sequentially to honor oss.Sequential()
	for _, chunk := range chunks {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			if ossToken, err = c.raw.GetOSSToken(); err != nil {
				return nil, err
			}
		default:
		}

		var part oss.UploadPart
		buf := make([]byte, chunk.Size)
		if _, err = f.ReadAt(buf, chunk.Offset); err != nil && err != io.EOF {
			return nil, err
		}

		for retry := 0; retry < 3; retry++ {
			if ctx.Err() != nil {
				return nil, ctx.Err()
			}
			part, err = bucket.UploadPart(
				imur,
				bytes.NewReader(buf),
				chunk.Size,
				chunk.Number,
				driver115.OssOption(params, ossToken)...,
			)
			if err == nil {
				break
			}
			time.Sleep(1 * time.Second) // Wait before retry
		}
		if err != nil {
			return nil, fmt.Errorf("failed to upload chunk %d after retries: %w", chunk.Number, err)
		}

		parts = append(parts, part)
		num := completedNum.Add(1)
		if progress != nil {
			progress(float64(num) * 100.0 / float64(len(chunks)))
		}
	}

	if _, err := bucket.CompleteMultipartUpload(imur, parts, append(
		driver115.OssOption(params, ossToken),
		oss.CallbackResult(&bodyBytes),
	)...); err != nil {
		return nil, err
	}

	var uploadResult driver115.UploadResult
	if err = json.Unmarshal(bodyBytes, &uploadResult); err != nil {
		return nil, err
	}
	return &uploadResult, uploadResult.Err(string(bodyBytes))
}

type fileChunk struct {
	Number int
	Offset int64
	Size   int64
}

type contextReader struct {
	ctx context.Context
	r   io.Reader
}

func (cr *contextReader) Read(p []byte) (int, error) {
	if err := cr.ctx.Err(); err != nil {
		return 0, err
	}
	return cr.r.Read(p)
}

type progressReader struct {
	r       io.Reader
	updater func(bytesRead int64)
	total   int64
}

func (pr *progressReader) Read(p []byte) (int, error) {
	n, err := pr.r.Read(p)
	if n > 0 {
		pr.total += int64(n)
		if pr.updater != nil {
			pr.updater(pr.total)
		}
	}
	return n, err
}

func convertFile(f *driver115.File) model115.File {
	return model115.File{
		ID:        f.FileID,
		Name:      f.Name,
		IsDir:     f.IsDirectory,
		Size:      f.Size,
		SHA1:      f.Sha1,
		PickCode:  f.PickCode,
		CreatedAt: f.CreateTime,
		UpdatedAt: f.UpdateTime,
	}
}
