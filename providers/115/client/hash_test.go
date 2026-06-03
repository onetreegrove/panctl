package client

import (
	"bytes"
	"context"
	"testing"
)

func TestHashUtils(t *testing.T) {
	data := []byte("hello world! this is a test string to calculate sha1 hash.")
	r := bytes.NewReader(data)

	// Test PreHash
	preVal, err := PreHash(context.Background(), r)
	if err != nil {
		t.Fatalf("PreHash failed: %v", err)
	}

	// Test FullHash
	var progressCount int64
	fullVal, err := FullHash(context.Background(), r, func(bytesRead int64) {
		progressCount = bytesRead
	})
	if err != nil {
		t.Fatalf("FullHash failed: %v", err)
	}

	if progressCount != int64(len(data)) {
		t.Errorf("expected progress %d, got %d", len(data), progressCount)
	}

	if preVal != fullVal {
		// Since data is short (< 128 KiB), preHash and fullHash should be equal!
		t.Errorf("expected equal hashes for small data, got pre=%s, full=%s", preVal, fullVal)
	}
}
