# Aliyun Provider Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add an Aliyun Drive provider that follows the existing `pan-cli` provider contract and supports refresh-token login, common file operations, download links, upload, and rapid upload.

**Architecture:** Use `tmp/drivers/aliyundrive_open` as the primary reference and keep all Aliyun-specific API code under `providers/aliyun`. Reuse the existing `providerOps` bridge in `internal/app`, the file credential store in `internal/credential`, and the path resolver in `internal/resolver`. Do not implement the deprecated non-open `tmp/drivers/aliyundrive` route or the share-only route in this phase.

**Tech Stack:** Go, Cobra, Resty, `golang.org/x/time/rate`, existing `internal/credential`, `internal/output`, `internal/resolver`, `pkg/contract`, and `pkg/provider`.

---

## File Structure

- Create `providers/aliyun/provider.go`: provider name and conservative capabilities.
- Create `providers/aliyun/model/credential.go`: refresh-token credential model, defaults, redaction, drive type normalization.
- Create `providers/aliyun/model/file.go`: Aliyun API file model and conversion to `contract.FileInfo`.
- Create `providers/aliyun/client/errors.go`: Aliyun remote errors to `contract.Error` mapping.
- Create `providers/aliyun/client/client.go`: Resty client, token refresh, request retry, access-token guard.
- Create `providers/aliyun/client/limiter.go`: per-user and global request limiters for list, link, and other API calls.
- Create `providers/aliyun/client/file.go`: list, download URL, mkdir, move, copy, rename, delete.
- Create `providers/aliyun/client/upload.go`: upload create, rapid upload proof, part upload, complete.
- Create `providers/aliyun/client/upload_math.go`: part-size and proof-code helpers isolated for unit tests.
- Create `cmd/aliyun-cli/main.go`: provider-specific entrypoint.
- Modify `internal/app/login.go`: add Aliyun refresh-token login command and flags.
- Modify `internal/app/provider_ops.go`: load Aliyun credentials and implement `opsAliyun`.
- Modify `internal/app/app.go`: avoid registering 115 share commands for non-115 providers.
- Modify `go.mod`: add `golang.org/x/time` only if it is not already available through existing imports.
- Add tests beside the created packages and extend `internal/app/app_test.go`.

---

### Task 1: Provider Contract, Credential, File Model, And Error Mapping

**Files:**
- Create: `providers/aliyun/provider.go`
- Create: `providers/aliyun/model/credential.go`
- Create: `providers/aliyun/model/file.go`
- Create: `providers/aliyun/client/errors.go`
- Test: `providers/aliyun/provider_test.go`
- Test: `providers/aliyun/model/credential_test.go`
- Test: `providers/aliyun/model/file_test.go`
- Test: `providers/aliyun/client/errors_test.go`

- [ ] **Step 1: Write failing provider capability test**

```go
package aliyun

import "testing"

func TestProviderCapabilities(t *testing.T) {
	caps := New().Capabilities()
	if !caps.PathLookup || !caps.List || !caps.Mkdir || !caps.Rename || !caps.Move || !caps.Copy || !caps.Remove || !caps.Download || !caps.Upload || !caps.RapidUpload {
		t.Fatalf("expected aliyun file capabilities to be enabled: %+v", caps)
	}
	if caps.Search || caps.OfflineTask || caps.Share || caps.RecycleBin || caps.CrossTransfer {
		t.Fatalf("expected unsupported aliyun capabilities to remain disabled: %+v", caps)
	}
	if New().Name() != "aliyun" {
		t.Fatalf("unexpected provider name: %s", New().Name())
	}
}
```

- [ ] **Step 2: Write failing credential tests**

```go
package model

import "testing"

func TestCredentialDefaultsAndRedaction(t *testing.T) {
	cred := Credential{RefreshToken: "abcdefghijklmnopqrstuvwxyz"}.WithDefaults()
	if cred.DriveType != DriveTypeResource {
		t.Fatalf("expected default drive type resource, got %q", cred.DriveType)
	}
	if got := cred.RedactedRefreshToken(); got != "abcd***wxyz" {
		t.Fatalf("unexpected redaction: %s", got)
	}
}

func TestCredentialNormalizesInvalidDriveType(t *testing.T) {
	cred := Credential{RefreshToken: "token", DriveType: "invalid"}.WithDefaults()
	if cred.DriveType != DriveTypeResource {
		t.Fatalf("invalid drive type should normalize to resource, got %q", cred.DriveType)
	}
}
```

- [ ] **Step 3: Write failing file conversion test**

```go
package model

import (
	"testing"
	"time"

	"github.com/justonetree/pan-cli/pkg/contract"
)

func TestFileToContract(t *testing.T) {
	created := time.Date(2026, 6, 2, 10, 0, 0, 0, time.UTC)
	updated := time.Date(2026, 6, 2, 11, 0, 0, 0, time.UTC)
	file := File{
		DriveID:      "drive_1",
		FileID:       "file_1",
		ParentFileID: "root",
		Name:         "demo.mp4",
		Size:         1024,
		Type:         "file",
		ContentHash:  "abc123",
		Thumbnail:    "https://thumb.example/demo.jpg",
		Category:     "video",
		CreatedAt:    created,
		UpdatedAt:    updated,
	}
	got := FromAPIFile(file).ToContract("/备份/demo.mp4")
	if got.ID != "file_1" || got.Name != "demo.mp4" || got.Type != contract.FileTypeFile || got.Provider != "aliyun" {
		t.Fatalf("unexpected contract file: %+v", got)
	}
	if got.Raw["drive_id"] != "drive_1" || got.Raw["content_hash"] != "abc123" {
		t.Fatalf("expected raw metadata, got %+v", got.Raw)
	}
}

func TestFolderToContract(t *testing.T) {
	got := FromAPIFile(File{FileID: "dir_1", Name: "资料", Type: "folder"}).ToContract("/资料")
	if got.Type != contract.FileTypeDir {
		t.Fatalf("expected dir, got %s", got.Type)
	}
}
```

- [ ] **Step 4: Write failing error mapping tests**

```go
package client

import (
	"errors"
	"testing"

	"github.com/justonetree/pan-cli/pkg/contract"
)

func TestMapError(t *testing.T) {
	cases := []struct {
		name string
		err  error
		code string
	}{
		{"auth", errors.New("AccessTokenExpired: token expired"), contract.CodeAuthExpired},
		{"not found", errors.New("FileNotFound: missing"), contract.CodeNotFound},
		{"permission", errors.New("Forbidden: denied"), contract.CodePermissionDenied},
		{"rate", errors.New("HTTP 429 too many requests"), contract.CodeRateLimited},
		{"network", errors.New("dial tcp timeout"), contract.CodeNetworkError},
		{"remote", errors.New("InvalidParameter: bad request"), contract.CodeRemoteError},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := MapError(tc.err); got.Code != tc.code {
				t.Fatalf("expected %s, got %+v", tc.code, got)
			}
		})
	}
}
```

- [ ] **Step 5: Run tests and verify they fail because packages are missing**

Run: `go test ./providers/aliyun/...`

Expected: FAIL with messages that `providers/aliyun` packages or identifiers do not exist.

- [ ] **Step 6: Implement provider, credential, file, and error mapping**

Use the tests above as the exact public behavior. Keep comments only where they explain Aliyun-specific decisions, such as why invalid drive types normalize to `resource`.

- [ ] **Step 7: Run focused tests**

Run: `go test ./providers/aliyun/...`

Expected: PASS.

- [ ] **Step 8: Commit**

```bash
git add providers/aliyun
git commit -m "feat: add aliyun provider contract models"
```

---

### Task 2: Aliyun HTTP Client, Token Refresh, And Limiters

**Files:**
- Create: `providers/aliyun/client/client.go`
- Create: `providers/aliyun/client/limiter.go`
- Test: `providers/aliyun/client/client_test.go`
- Test: `providers/aliyun/client/limiter_test.go`

- [ ] **Step 1: Write failing token refresh test with fake HTTP server**

```go
func TestRefreshTokenSelectsResourceDrive(t *testing.T) {
	var tokenCalled bool
	var driveCalled bool
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/access_token":
			tokenCalled = true
			_, _ = w.Write([]byte(`{"access_token":"access_1","refresh_token":"refresh_2"}`))
		case "/adrive/v1.0/user/getDriveInfo":
			driveCalled = true
			_, _ = w.Write([]byte(`{"user_id":"user_1","default_drive_id":"default_1","resource_drive_id":"resource_1","backup_drive_id":"backup_1"}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	cred := model.Credential{RefreshToken: "refresh_1", ClientID: "cid", ClientSecret: "secret", DriveType: model.DriveTypeResource}.WithDefaults()
	c := New(Options{APIBaseURL: srv.URL, OAuthTokenURL: srv.URL + "/oauth/access_token", RequestsPerSecond: 0})
	c.ImportCredential(cred)
	if err := c.RefreshToken(context.Background()); err != nil {
		t.Fatalf("RefreshToken returned error: %v", err)
	}
	if !tokenCalled || !driveCalled {
		t.Fatalf("expected token and drive endpoints to be called")
	}
	got := c.Credential()
	if got.AccessToken != "access_1" || got.RefreshToken != "refresh_2" || got.DriveID != "resource_1" || got.UserID != "user_1" {
		t.Fatalf("unexpected credential: %+v", got)
	}
}
```

- [ ] **Step 2: Write failing request retry test for expired access token**

```go
func TestRequestRefreshesExpiredAccessTokenOnce(t *testing.T) {
	var protectedCalls int
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/oauth/access_token":
			_, _ = w.Write([]byte(`{"access_token":"access_2","refresh_token":"refresh_2"}`))
		case "/adrive/v1.0/user/getDriveInfo":
			_, _ = w.Write([]byte(`{"user_id":"user_1","resource_drive_id":"resource_1"}`))
		case "/protected":
			protectedCalls++
			if protectedCalls == 1 {
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"code":"AccessTokenExpired","message":"expired"}`))
				return
			}
			_, _ = w.Write([]byte(`{"ok":true}`))
		default:
			http.NotFound(w, r)
		}
	}))
	defer srv.Close()

	c := New(Options{APIBaseURL: srv.URL, OAuthTokenURL: srv.URL + "/oauth/access_token"})
	c.ImportCredential(model.Credential{RefreshToken: "refresh_1", AccessToken: "access_1", DriveID: "resource_1", UserID: "user_1"}.WithDefaults())
	var out map[string]bool
	if _, err := c.Request(context.Background(), limiterOther, "/protected", http.MethodPost, nil, &out); err != nil {
		t.Fatalf("request returned error: %v", err)
	}
	if protectedCalls != 2 || !out["ok"] {
		t.Fatalf("expected retry success, calls=%d out=%+v", protectedCalls, out)
	}
}
```

- [ ] **Step 3: Write limiter cancellation test**

```go
func TestLimiterWaitHonorsContextCancellation(t *testing.T) {
	lim := newUserLimiter(0.001)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := lim.wait(ctx, limiterList); err == nil {
		t.Fatalf("expected canceled context error")
	}
}
```

- [ ] **Step 4: Implement client and limiter**

Implement:
- `type Options struct { APIBaseURL string; OAuthTokenURL string; RequestsPerSecond float64 }`
- `func New(opts Options) *Client`
- `func (c *Client) ImportCredential(model.Credential)`
- `func (c *Client) Credential() model.Credential`
- `func (c *Client) RefreshToken(ctx context.Context) error`
- `func (c *Client) Request(ctx context.Context, typ limiterType, uri, method string, body any, result any) ([]byte, error)`

Use `https://openapi.alipan.com` as the default API base URL. Use `https://openapi.alipan.com/oauth/access_token` as the default direct token URL unless `OAuthTokenURL` is provided. Refresh on `AccessTokenInvalid`, `AccessTokenExpired`, and `I400JD`, then retry the original request once.

- [ ] **Step 5: Run focused tests**

Run: `go test ./providers/aliyun/client -run 'TestRefreshToken|TestRequest|TestLimiter'`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add providers/aliyun/client providers/aliyun/model
git commit -m "feat: add aliyun openapi client"
```

---

### Task 3: File Operations Client

**Files:**
- Create: `providers/aliyun/client/file.go`
- Test: `providers/aliyun/client/file_test.go`

- [ ] **Step 1: Write fake-server tests for list and download URL**

```go
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
```

- [ ] **Step 2: Write fake-server tests for mutate operations**

```go
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
```

- [ ] **Step 3: Implement file operations**

Implement:
- `type ListResult struct { Items []model.File; Total int; HasMore bool; NextPage int }`
- `func (c *Client) List(ctx context.Context, parentFileID string, page, limit int) (ListResult, error)`
- `func (c *Client) DownloadURL(ctx context.Context, fileID string) (string, map[string][]string, error)`
- `func (c *Client) Mkdir(ctx context.Context, parentID, name string) (model.File, error)`
- `func (c *Client) Move(ctx context.Context, destID string, fileIDs ...string) error`
- `func (c *Client) Copy(ctx context.Context, destID string, fileIDs ...string) error`
- `func (c *Client) Rename(ctx context.Context, fileID, newName string) error`
- `func (c *Client) Delete(ctx context.Context, fileIDs ...string) error`

For `List`, request `limit` and `marker`. If `page > 1`, iterate markers internally until the requested page. Set `HasMore` from non-empty `next_marker`.

- [ ] **Step 4: Run focused tests**

Run: `go test ./providers/aliyun/client -run 'TestList|TestMkdir'`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add providers/aliyun/client/file.go providers/aliyun/client/file_test.go
git commit -m "feat: add aliyun file operations"
```

---

### Task 4: Upload Helpers And Upload Client

**Files:**
- Create: `providers/aliyun/client/upload_math.go`
- Create: `providers/aliyun/client/upload.go`
- Test: `providers/aliyun/client/upload_math_test.go`
- Test: `providers/aliyun/client/upload_test.go`

- [ ] **Step 1: Write failing helper tests**

```go
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
```

- [ ] **Step 2: Write failing normal upload fake-server test**

```go
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
```

- [ ] **Step 3: Write failing rapid upload fake-server test**

```go
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
```

- [ ] **Step 4: Implement upload helpers and upload client**

Implement:
- `func PartSize(fileSize int64) int64`
- `func ProofCode(accessToken string, size int64, r io.ReaderAt) (string, error)`
- `func SHA1Hex(ctx context.Context, r io.Reader) (string, error)`
- `func PreHash(ctx context.Context, r io.Reader) (string, error)`
- `func (c *Client) Upload(ctx context.Context, localPath, destDirID string, progress func(float64)) (model.File, error)`

Keep normal upload and rapid upload in one method because the remote create flow decides which branch succeeds. Refresh upload URLs when elapsed upload time exceeds 50 minutes.

- [ ] **Step 5: Run focused tests**

Run: `go test ./providers/aliyun/client -run 'TestPartSize|TestProofCode|TestUpload'`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add providers/aliyun/client/upload*.go providers/aliyun/client/upload*_test.go
git commit -m "feat: add aliyun upload"
```

---

### Task 5: CLI Login And Provider-Specific Entry Point

**Files:**
- Create: `cmd/aliyun-cli/main.go`
- Modify: `internal/app/login.go`
- Test: `internal/app/app_test.go`

- [ ] **Step 1: Write failing CLI smoke tests**

Add tests that run:

```bash
go run ../../cmd/aliyun-cli --json --config-dir <temp> login status
go run ../../cmd/aliyun-cli --json --config-dir <temp> login refresh-token --token refresh-token-value --skip-check
```

Expected JSON for status before login:

```json
{
  "status": "ok",
  "meta": { "provider": "aliyun", "profile": "default" },
  "data": { "authenticated": false }
}
```

Expected JSON for skip-check login:

```json
{
  "status": "ok",
  "meta": { "provider": "aliyun", "profile": "default" },
  "data": {
    "authenticated": true,
    "refresh_token": "refr***alue"
  }
}
```

- [ ] **Step 2: Implement `cmd/aliyun-cli/main.go`**

```go
package main

import (
	"os"

	"github.com/justonetree/pan-cli/internal/app"
)

func main() {
	os.Exit(app.Run(app.Options{BinaryName: "aliyun-cli", DefaultProvider: "aliyun"}))
}
```

- [ ] **Step 3: Extend login command for Aliyun**

Refactor the existing Baidu-only `loginRefreshTokenCommand` into provider-aware branches or create `loginAliyunRefreshTokenCommand`. Keep Baidu behavior unchanged. Aliyun flags must include `--client-id`, `--client-secret`, `--oauth-token-url`, `--drive-type`, and `--skip-check`.

- [ ] **Step 4: Run CLI tests**

Run: `go test ./internal/app -run 'TestAliyun'`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add cmd/aliyun-cli internal/app/login.go internal/app/app_test.go
git commit -m "feat: add aliyun login cli"
```

---

### Task 6: Wire Aliyun Into Common File Commands

**Files:**
- Modify: `internal/app/provider_ops.go`
- Modify: `internal/app/app.go`
- Test: `internal/app/app_test.go`

- [ ] **Step 1: Write failing provider selection test**

Add a test that stores an Aliyun credential JSON in a temp config dir and runs:

```bash
go run ../../cmd/aliyun-cli --json --config-dir <temp> ls /
```

Use a fake Aliyun API server by injecting test URLs through credential fields or client options. If process-level injection is too invasive, test `getProviderOps` directly inside `internal/app` package and assert it returns an Aliyun ops implementation for `DefaultProvider: "aliyun"`.

- [ ] **Step 2: Implement `opsAliyun`**

Add to `internal/app/provider_ops.go`:
- load `modelAliyun.Credential`
- create `clientAliyun.New(clientAliyun.Options{})`
- import credential
- implement `List`, `ListChildren`, `DownloadURL`, `Mkdir`, `Move`, `Copy`, `Rename`, `Delete`, and `Upload`

Use ID-based operations like 115. For paths, rely on `internal/resolver` to call `ListChildren` from root.

- [ ] **Step 3: Guard 115 share commands**

Modify `internal/app/app.go` so 115 share commands are added only when `rt.providerName() == "115"`. This prevents `aliyun-cli share ...` from accidentally binding to 115-only command handlers.

- [ ] **Step 4: Run app tests**

Run: `go test ./internal/app`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/app/provider_ops.go internal/app/app.go internal/app/app_test.go
git commit -m "feat: wire aliyun provider operations"
```

---

### Task 7: Registry, Builds, And Documentation Sync

**Files:**
- Modify: `pkg/provider/registry_test.go`
- Modify: `pkg/provider/registry.go` only if the registry currently hardcodes provider names.
- Modify: `README.md` if provider list is present.
- Modify: `docs/providers/aliyun.md` only if implementation behavior differs from the current design document.

- [ ] **Step 1: Add registry test for Aliyun**

If the registry has provider registration tests, add an assertion that `aliyun` can be registered and queried without changing existing 115 and Baidu behavior.

- [ ] **Step 2: Update docs only to match implemented behavior**

Keep [docs/providers/aliyun.md](/Users/justonetree/wwwroot/pan/docs/providers/aliyun.md) aligned with real flags, capabilities, and remove mode. Do not add share or search commands in this phase.

- [ ] **Step 3: Run full tests**

Run: `go test ./...`

Expected: PASS.

- [ ] **Step 4: Build binaries**

Run: `go build ./cmd/pan ./cmd/115-cli ./cmd/baidu-cli ./cmd/aliyun-cli`

Expected: command exits 0 and produces no compiler errors.

- [ ] **Step 5: Commit**

```bash
git add pkg/provider README.md docs/providers/aliyun.md
git commit -m "docs: sync aliyun provider implementation notes"
```

---

### Task 8: Manual Smoke With Real Account

**Files:**
- Modify only if smoke reveals a defect in implementation or docs.

- [ ] **Step 1: Login with real refresh token**

Run:

```bash
aliyun-cli --json login refresh-token --token "$ALIYUN_REFRESH_TOKEN"
```

Expected: JSON `status=ok`, `data.authenticated=true`, redacted refresh token.

- [ ] **Step 2: Exercise read commands**

Run:

```bash
aliyun-cli --json ls /
aliyun-cli --json download /path/to/existing-file
```

Expected: list returns `items`; download returns a URL, headers, and an `expires_at` value.

- [ ] **Step 3: Exercise write commands in a disposable directory**

Run:

```bash
aliyun-cli --json mkdir / pan-cli-smoke
aliyun-cli --json upload ./README.md --to /pan-cli-smoke
aliyun-cli --json rename /pan-cli-smoke/README.md README-renamed.md
aliyun-cli --json cp /pan-cli-smoke/README-renamed.md --to /
aliyun-cli --json mv /README-renamed.md --to /pan-cli-smoke
aliyun-cli --json rm /pan-cli-smoke/README-renamed.md
```

Expected: every command returns `status=ok`. The disposable directory can remain for post-run inspection or be removed by the final `rm` commands.

- [ ] **Step 4: Document smoke result**

Record the exact commands, timestamps, and whether rapid upload was hit in a short note under `docs/testing/aliyun-smoke.md`.

- [ ] **Step 5: Commit smoke doc**

```bash
git add docs/testing/aliyun-smoke.md
git commit -m "test: document aliyun provider smoke"
```

---

## Final Verification

- [ ] Run `gofmt` on all touched Go files.
- [ ] Run `go test ./...`.
- [ ] Run `go build ./cmd/pan ./cmd/115-cli ./cmd/baidu-cli ./cmd/aliyun-cli`.
- [ ] Confirm `git status --short` contains only intended files before final handoff.

## Self-Review Notes

- Spec coverage: refresh-token auth, drive type selection, common provider operations, upload, rapid upload, download, limiter, error mapping, CLI entrypoint, and tests are covered.
- Out of scope: search, offline tasks, recycle bin management, share creation, share read commands, deprecated non-open route, and cross-provider transfer.
- Type consistency: implementation names use `providers/aliyun`, `model.Credential`, `model.File`, `client.Client`, and `opsAliyun` consistently.
