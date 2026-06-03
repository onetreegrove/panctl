package client

import (
	"context"
	"crypto/sha1"
	"encoding/hex"
	"io"
)

type HashProgressUpdater func(bytesRead int64)

// PreHash calculates the SHA1 of the first 128 KiB (131,072 bytes) of the reader.
// It will attempt to seek back to the starting offset of the ReadSeeker if successful.
func PreHash(ctx context.Context, r io.ReadSeeker) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	// Save start position
	startPos, err := r.Seek(0, io.SeekCurrent)
	if err != nil {
		return "", err
	}

	h := sha1.New()
	// Read at most 128 KiB
	limitReader := io.LimitReader(r, 128*1024)

	// Read in loop to allow context cancellation
	buf := make([]byte, 32*1024)
	for {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}
		n, err := limitReader.Read(buf)
		if n > 0 {
			if _, wErr := h.Write(buf[:n]); wErr != nil {
				return "", wErr
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
	}

	// Seek back to start position
	if _, err := r.Seek(startPos, io.SeekStart); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}

// FullHash calculates the SHA1 of the entire content of the ReadSeeker.
// It will attempt to seek back to the starting offset of the ReadSeeker if successful.
func FullHash(ctx context.Context, r io.ReadSeeker, updater HashProgressUpdater) (string, error) {
	if ctx.Err() != nil {
		return "", ctx.Err()
	}

	// Save start position
	startPos, err := r.Seek(0, io.SeekCurrent)
	if err != nil {
		return "", err
	}

	h := sha1.New()
	buf := make([]byte, 32*1024)
	var totalRead int64

	for {
		if ctx.Err() != nil {
			return "", ctx.Err()
		}

		n, err := r.Read(buf)
		if n > 0 {
			if _, wErr := h.Write(buf[:n]); wErr != nil {
				return "", wErr
			}
			totalRead += int64(n)
			if updater != nil {
				updater(totalRead)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}
	}

	// Seek back to start position
	if _, err := r.Seek(startPos, io.SeekStart); err != nil {
		return "", err
	}

	return hex.EncodeToString(h.Sum(nil)), nil
}
