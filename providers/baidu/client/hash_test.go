package client

import (
	"bytes"
	"context"
	"testing"
)

func TestMD5HexPreservesReaderPosition(t *testing.T) {
	r := bytes.NewReader([]byte("hello"))

	got, err := MD5Hex(context.Background(), r)
	if err != nil {
		t.Fatal(err)
	}
	if got != "5d41402abc4b2a76b9719d911017c592" {
		t.Fatalf("md5 = %q", got)
	}
	pos, err := r.Seek(0, 1)
	if err != nil {
		t.Fatal(err)
	}
	if pos != 0 {
		t.Fatalf("reader position = %d", pos)
	}
}

func TestSliceMD5s(t *testing.T) {
	r := bytes.NewReader([]byte("abcdef"))

	got, err := SliceMD5s(context.Background(), r, 3)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"900150983cd24fb0d6963f7d28e17f72", "4ed9407630eb1000c0f6b63842defa7d"}
	if len(got) != len(want) {
		t.Fatalf("len = %d", len(got))
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("md5[%d] = %q, want %q", i, got[i], want[i])
		}
	}
}
