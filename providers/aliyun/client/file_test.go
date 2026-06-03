package client

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/onetreegrove/panctl/providers/aliyun/model"
)

func newTestClient(baseURL string) *Client {
	c := New(Options{APIBaseURL: baseURL, OAuthTokenURL: baseURL + "/oauth/access_token", RequestsPerSecond: 1000})
	c.ImportCredential(model.Credential{
		RefreshToken: "refresh_1",
		AccessToken:  "access_1",
		DriveID:      "drive_1",
		UserID:       "user_1",
	}.WithDefaults())
	return c
}

func TestListAndDownloadURL(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/adrive/v1.0/openFile/list":
			_, _ = w.Write([]byte(`{"items":[{"drive_id":"drive_1","file_id":"file_1","parent_file_id":"root","name":"demo.txt","type":"file","size":12,"content_hash":"sha1_1","updated_at":"2026-06-02T10:00:00Z","created_at":"2026-06-02T09:00:00Z"}],"next_marker":""}`))
		case "/adrive/v1.0/openFile/getDownloadUrl":
			_, _ = w.Write([]byte(`{"url":"https://download.example/demo.txt"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	res, err := c.List(context.Background(), "root", 1, 100)
	if err != nil {
		t.Fatalf("List returned error: %v", err)
	}
	if len(res.Items) != 1 || res.Items[0].ID != "file_1" || res.HasMore {
		t.Fatalf("unexpected list result: %+v", res)
	}
	url, headers, err := c.DownloadURL(context.Background(), "file_1")
	if err != nil {
		t.Fatalf("DownloadURL returned error: %v", err)
	}
	if url != "https://download.example/demo.txt" || headers["Referer"][0] != "https://www.alipan.com/" {
		t.Fatalf("unexpected download result: %s %+v", url, headers)
	}
}

func TestMkdirMoveCopyRenameDelete(t *testing.T) {
	seen := map[string]int{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen[r.URL.Path]++
		switch r.URL.Path {
		case "/adrive/v1.0/openFile/create":
			_, _ = w.Write([]byte(`{"drive_id":"drive_1","file_id":"dir_1","parent_file_id":"root","name":"资料","type":"folder","created_at":"2026-06-02T09:00:00Z","updated_at":"2026-06-02T09:00:00Z"}`))
		case "/adrive/v1.0/openFile/move", "/adrive/v1.0/openFile/copy", "/adrive/v1.0/openFile/update", "/adrive/v1.0/openFile/recyclebin/trash":
			_, _ = w.Write([]byte(`{"file_id":"file_1"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := newTestClient(srv.URL)
	if _, err := c.Mkdir(context.Background(), "root", "资料"); err != nil {
		t.Fatalf("Mkdir returned error: %v", err)
	}
	if err := c.Move(context.Background(), "dir_1", "file_1"); err != nil {
		t.Fatalf("Move returned error: %v", err)
	}
	if err := c.Copy(context.Background(), "dir_1", "file_1"); err != nil {
		t.Fatalf("Copy returned error: %v", err)
	}
	if err := c.Rename(context.Background(), "file_1", "new.txt"); err != nil {
		t.Fatalf("Rename returned error: %v", err)
	}
	if err := c.Delete(context.Background(), "file_1"); err != nil {
		t.Fatalf("Delete returned error: %v", err)
	}
	if seen["/adrive/v1.0/openFile/create"] != 1 || seen["/adrive/v1.0/openFile/recyclebin/trash"] != 1 {
		t.Fatalf("unexpected endpoint calls: %+v", seen)
	}
}
