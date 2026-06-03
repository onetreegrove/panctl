package client

import (
	"context"
	"crypto/md5"
	"crypto/sha1"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
)

const (
	kb = 1024
	mb = 1024 * kb
	gb = 1024 * mb
	tb = 1024 * gb
)

func PartSize(fileSize int64) int64 {
	partSize := int64(20 * mb)
	if fileSize <= partSize {
		return partSize
	}
	switch {
	case fileSize > 1*tb:
		return 5 * gb
	case fileSize > 768*gb:
		return 109951163
	case fileSize > 512*gb:
		return 82463373
	case fileSize > 384*gb:
		return 54975582
	case fileSize > 256*gb:
		return 41231687
	case fileSize > 128*gb:
		return 27487791
	default:
		return partSize
	}
}

func ProofCode(accessToken string, size int64, r io.ReaderAt) (string, error) {
	if size <= 0 {
		return "", nil
	}
	sum := md5.Sum([]byte(accessToken))
	hashHex := hex.EncodeToString(sum[:])
	index, err := strconv.ParseUint(hashHex[:16], 16, 64)
	if err != nil {
		return "", err
	}
	start := int64(index % uint64(size))
	length := int64(8)
	if start+length > size {
		length = size - start
	}
	buf := make([]byte, length)
	n, err := r.ReadAt(buf, start)
	if err != nil && err != io.EOF {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf[:n]), nil
}

func SHA1Hex(ctx context.Context, r io.Reader) (string, error) {
	h := sha1.New()
	buf := make([]byte, 128*kb)
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

func PreHash(ctx context.Context, r io.Reader) (string, error) {
	lr := io.LimitReader(r, 1024)
	hash, err := SHA1Hex(ctx, lr)
	if err != nil {
		return "", err
	}
	if hash == "" {
		return "", fmt.Errorf("empty pre hash")
	}
	return hash, nil
}
