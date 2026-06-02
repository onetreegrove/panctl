package client

import (
	"context"
	"crypto/md5"
	"encoding/hex"
	"io"
)

func MD5Hex(ctx context.Context, r io.ReadSeeker) (string, error) {
	start, err := r.Seek(0, io.SeekCurrent)
	if err != nil {
		return "", err
	}
	defer r.Seek(start, io.SeekStart)

	h := md5.New()
	buf := make([]byte, 32*1024)
	for {
		if err := ctx.Err(); err != nil {
			return "", err
		}
		n, err := r.Read(buf)
		if n > 0 {
			if _, writeErr := h.Write(buf[:n]); writeErr != nil {
				return "", writeErr
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
	}
	return hex.EncodeToString(h.Sum(nil)), nil
}

func SliceMD5s(ctx context.Context, r io.ReadSeeker, sliceSize int64) ([]string, error) {
	start, err := r.Seek(0, io.SeekCurrent)
	if err != nil {
		return nil, err
	}
	defer r.Seek(start, io.SeekStart)

	var result []string
	buf := make([]byte, sliceSize)
	for {
		if err := ctx.Err(); err != nil {
			return nil, err
		}
		n, err := io.ReadFull(r, buf)
		if n > 0 {
			h := md5.Sum(buf[:n])
			result = append(result, hex.EncodeToString(h[:]))
		}
		if err == io.EOF || err == io.ErrUnexpectedEOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}
	return result, nil
}
