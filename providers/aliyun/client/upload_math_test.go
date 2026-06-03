package client

import (
	"bytes"
	"testing"
)

func TestPartSize(t *testing.T) {
	if got := PartSize(10 * 1024 * 1024); got != 20*1024*1024 {
		t.Fatalf("small file part size = %d", got)
	}
	if got := PartSize(900 * 1024 * 1024 * 1024); got <= 20*1024*1024 {
		t.Fatalf("large file should use larger part size, got %d", got)
	}
}

func TestProofCode(t *testing.T) {
	data := []byte("0123456789abcdefghijklmnopqrstuvwxyz")
	code, err := ProofCode("access-token", int64(len(data)), bytes.NewReader(data))
	if err != nil {
		t.Fatalf("ProofCode returned error: %v", err)
	}
	if code == "" {
		t.Fatalf("expected non-empty proof code")
	}
}
