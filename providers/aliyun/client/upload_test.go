package client

import (
	"bytes"
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
)

func TestUploadNormalFlow(t *testing.T) {
	tmp := t.TempDir()
	local := filepath.Join(tmp, "demo.txt")
	if err := os.WriteFile(local, []byte("hello aliyun upload"), 0600); err != nil {
		t.Fatal(err)
	}
	var putCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/adrive/v1.0/openFile/create":
			_, _ = w.Write([]byte(`{"file_id":"file_1","upload_id":"upload_1","rapid_upload":false,"part_info_list":[{"part_number":1,"upload_url":"` + "http://" + r.Host + `/upload-part"}]}`))
		case "/upload-part":
			putCalled = true
			w.WriteHeader(http.StatusOK)
		case "/adrive/v1.0/openFile/complete":
			_, _ = w.Write([]byte(`{"drive_id":"drive_1","file_id":"file_1","parent_file_id":"root","name":"demo.txt","type":"file","size":19,"created_at":"2026-06-02T09:00:00Z","updated_at":"2026-06-02T09:00:00Z"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	file, err := c.Upload(context.Background(), local, "root", nil)
	if err != nil {
		t.Fatalf("Upload returned error: %v", err)
	}
	if !putCalled || file.ID != "file_1" {
		t.Fatalf("unexpected upload result put=%v file=%+v", putCalled, file)
	}
}

func TestUploadRapidFlow(t *testing.T) {
	tmp := t.TempDir()
	local := filepath.Join(tmp, "rapid.bin")
	if err := os.WriteFile(local, bytes.Repeat([]byte("a"), 128*1024), 0600); err != nil {
		t.Fatal(err)
	}
	createCalls := 0
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/adrive/v1.0/openFile/create":
			createCalls++
			if createCalls == 1 {
				_, _ = w.Write([]byte(`{"code":"PreHashMatched","message":"pre hash matched"}`))
				return
			}
			_, _ = w.Write([]byte(`{"file_id":"file_rapid","upload_id":"upload_rapid","rapid_upload":true,"part_info_list":[]}`))
		case "/adrive/v1.0/openFile/complete":
			_, _ = w.Write([]byte(`{"drive_id":"drive_1","file_id":"file_rapid","parent_file_id":"root","name":"rapid.bin","type":"file","size":131072,"created_at":"2026-06-02T09:00:00Z","updated_at":"2026-06-02T09:00:00Z"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	file, err := c.Upload(context.Background(), local, "root", nil)
	if err != nil {
		t.Fatalf("Upload returned error: %v", err)
	}
	if createCalls != 2 || file.ID != "file_rapid" {
		t.Fatalf("unexpected rapid upload createCalls=%d file=%+v", createCalls, file)
	}
}
