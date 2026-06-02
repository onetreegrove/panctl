# 115 Provider Phase 1 Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the first usable `pan-cli` implementation with a 115 provider that supports cookie login, status, JSON output, listing, path resolution, file mutations, download links, and offline tasks.

**Architecture:** Implement a small Go monorepo with shared contracts, output/error handling, config/credential storage, provider registry, and a 115 provider backed by `github.com/SheltonZhu/115driver`. Keep provider-specific API details under `providers/115`, while `cmd/*` and `internal/app` only orchestrate commands and stable contracts.

**Tech Stack:** Go 1.22+, Cobra CLI, `github.com/SheltonZhu/115driver` with Alist-compatible replace if needed, standard `encoding/json`, `os.UserConfigDir`, and Go tests.

---

## File Structure

Create this structure during implementation:

```text
cmd/
├── pan/main.go
└── 115-cli/main.go

internal/
├── app/
│   ├── app.go
│   ├── files.go
│   ├── login.go
│   └── offline.go
├── config/
│   ├── config.go
│   └── config_test.go
├── credential/
│   ├── file_store.go
│   └── file_store_test.go
├── output/
│   ├── json.go
│   ├── errors.go
│   └── json_test.go
└── resolver/
    ├── resolver.go
    └── resolver_test.go

pkg/
├── contract/
│   ├── error.go
│   ├── file.go
│   ├── offline.go
│   └── response.go
└── provider/
    ├── provider.go
    └── registry.go

providers/
└── 115/
    ├── provider.go
    ├── auth/
    │   ├── cookie.go
    │   └── cookie_test.go
    ├── client/
    │   ├── appver.go
    │   ├── client.go
    │   ├── errors.go
    │   ├── file.go
    │   └── offline.go
    └── model/
        ├── credential.go
        ├── file.go
        ├── file_test.go
        ├── offline.go
        └── offline_test.go
```

Do not implement upload, search, share, rapid-upload, recycle bin, or cross-provider transfer in this plan.

---

### Task 1: Go Module And Dependency Scaffold

**Files:**
- Create: `go.mod`
- Create: `cmd/pan/main.go`
- Create: `cmd/115-cli/main.go`

- [ ] **Step 1: Create `go.mod`**

Use this initial module definition:

```go
module github.com/justonetree/pan-cli

go 1.22

require (
	github.com/SheltonZhu/115driver v1.2.3-1
	github.com/spf13/cobra v1.8.1
	golang.org/x/time v0.5.0
)

replace github.com/SheltonZhu/115driver => github.com/okatu-loli/115driver v1.2.3-1
```

- [ ] **Step 2: Create minimal entrypoints**

`cmd/pan/main.go`:

```go
package main

import (
	"os"

	"github.com/justonetree/pan-cli/internal/app"
)

func main() {
	os.Exit(app.Run(app.Options{BinaryName: "pan"}))
}
```

`cmd/115-cli/main.go`:

```go
package main

import (
	"os"

	"github.com/justonetree/pan-cli/internal/app"
)

func main() {
	os.Exit(app.Run(app.Options{BinaryName: "115-cli", DefaultProvider: "115"}))
}
```

- [ ] **Step 3: Run module tidy**

Run: `go mod tidy`

Expected: `go.sum` is created and dependency resolution succeeds.

- [ ] **Step 4: Commit**

```bash
git add go.mod go.sum cmd/pan/main.go cmd/115-cli/main.go
git commit -m "chore: scaffold pan cli module"
```

---

### Task 2: Contracts And JSON Output

**Files:**
- Create: `pkg/contract/error.go`
- Create: `pkg/contract/file.go`
- Create: `pkg/contract/offline.go`
- Create: `pkg/contract/response.go`
- Create: `internal/output/json.go`
- Create: `internal/output/errors.go`
- Create: `internal/output/json_test.go`

- [ ] **Step 1: Write output tests**

`internal/output/json_test.go`:

```go
package output

import (
	"bytes"
	"encoding/json"
	"testing"

	"github.com/justonetree/pan-cli/pkg/contract"
)

func TestWriteOKWritesSingleJSONToStdout(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := WriteOK(&stdout, &stderr, contract.Meta{
		Provider:  "115",
		Profile:   "default",
		RequestID: "req_test",
	}, map[string]string{"hello": "world"})

	if code != 0 {
		t.Fatalf("exit code = %d, want 0", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	var got map[string]any
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("stdout is not JSON: %v", err)
	}
	if got["status"] != "ok" {
		t.Fatalf("status = %v, want ok", got["status"])
	}
}

func TestWriteErrorReturnsNonZeroAndJSONError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	code := WriteError(&stdout, &stderr, contract.Meta{
		Provider:  "115",
		Profile:   "default",
		RequestID: "req_test",
	}, contract.NewError(contract.CodeAuthRequired, "missing auth", "run login", false))

	if code != 10 {
		t.Fatalf("exit code = %d, want 10", code)
	}
	if stderr.Len() != 0 {
		t.Fatalf("stderr = %q, want empty", stderr.String())
	}

	var got struct {
		Status string `json:"status"`
		Error  struct {
			Code string `json:"code"`
		} `json:"error"`
	}
	if err := json.Unmarshal(stdout.Bytes(), &got); err != nil {
		t.Fatalf("stdout is not JSON: %v", err)
	}
	if got.Status != "error" || got.Error.Code != "AUTH_REQUIRED" {
		t.Fatalf("unexpected error response: %+v", got)
	}
}
```

- [ ] **Step 2: Verify tests fail**

Run: `go test ./internal/output`

Expected: FAIL because `WriteOK`, `WriteError`, and contract types are not defined.

- [ ] **Step 3: Add contract types**

`pkg/contract/error.go`:

```go
package contract

type ErrorCode string

const (
	CodeUsageError            ErrorCode = "USAGE_ERROR"
	CodeAuthRequired          ErrorCode = "AUTH_REQUIRED"
	CodeAuthExpired           ErrorCode = "AUTH_EXPIRED"
	CodePermissionDenied      ErrorCode = "PERMISSION_DENIED"
	CodeNotFound              ErrorCode = "NOT_FOUND"
	CodeConflict              ErrorCode = "CONFLICT"
	CodeUnsupportedCapability ErrorCode = "UNSUPPORTED_CAPABILITY"
	CodeRateLimited           ErrorCode = "RATE_LIMITED"
	CodeNetworkError          ErrorCode = "NETWORK_ERROR"
	CodeRemoteError           ErrorCode = "REMOTE_ERROR"
	CodeInternalError         ErrorCode = "INTERNAL_ERROR"
)

type Error struct {
	Code      ErrorCode `json:"code"`
	Message   string    `json:"message"`
	Detail    string    `json:"detail,omitempty"`
	Retryable bool      `json:"retryable"`
}

func NewError(code ErrorCode, message, detail string, retryable bool) Error {
	return Error{Code: code, Message: message, Detail: detail, Retryable: retryable}
}

func ExitCode(code ErrorCode) int {
	switch code {
	case CodeUsageError:
		return 2
	case CodeAuthRequired:
		return 10
	case CodeAuthExpired:
		return 11
	case CodePermissionDenied:
		return 12
	case CodeNotFound:
		return 20
	case CodeConflict:
		return 21
	case CodeUnsupportedCapability:
		return 22
	case CodeRateLimited:
		return 30
	case CodeNetworkError:
		return 31
	case CodeRemoteError:
		return 40
	default:
		return 50
	}
}
```

`pkg/contract/response.go`:

```go
package contract

type Meta struct {
	Provider  string `json:"provider,omitempty"`
	Profile   string `json:"profile"`
	RequestID string `json:"request_id"`
}

type Response struct {
	Status     string      `json:"status"`
	Data       any         `json:"data,omitempty"`
	Error      *Error      `json:"error,omitempty"`
	Pagination *Pagination `json:"pagination,omitempty"`
	Meta       Meta        `json:"meta"`
}

type Pagination struct {
	Page     int  `json:"page"`
	Limit    int  `json:"limit"`
	Total    int  `json:"total,omitempty"`
	HasMore  bool `json:"has_more"`
	NextPage int  `json:"next_page,omitempty"`
}
```

`pkg/contract/file.go`:

```go
package contract

import "time"

type FileType string

const (
	FileTypeFile FileType = "file"
	FileTypeDir  FileType = "dir"
)

type FileInfo struct {
	ID        string         `json:"id"`
	Name      string         `json:"name"`
	Type      FileType       `json:"type"`
	Path      string         `json:"path,omitempty"`
	Size      int64          `json:"size"`
	SHA1      string         `json:"sha1,omitempty"`
	PickCode  string         `json:"pick_code,omitempty"`
	ThumbURL  string         `json:"thumb_url,omitempty"`
	CreatedAt *time.Time     `json:"created_at,omitempty"`
	UpdatedAt *time.Time     `json:"updated_at,omitempty"`
	Provider  string         `json:"provider"`
	Raw       map[string]any `json:"raw,omitempty"`
}
```

`pkg/contract/offline.go`:

```go
package contract

type OfflineStatus string

const (
	OfflinePending OfflineStatus = "pending"
	OfflineRunning OfflineStatus = "running"
	OfflineDone    OfflineStatus = "done"
	OfflineFailed  OfflineStatus = "failed"
	OfflineUnknown OfflineStatus = "unknown"
)

type OfflineTask struct {
	GID      string         `json:"gid"`
	Name     string         `json:"name,omitempty"`
	Status   OfflineStatus  `json:"status"`
	Progress float64        `json:"progress"`
	Size     int64          `json:"size,omitempty"`
	FileID   string         `json:"file_id,omitempty"`
	Raw      map[string]any `json:"raw,omitempty"`
}
```

- [ ] **Step 4: Add output implementation**

`internal/output/json.go`:

```go
package output

import (
	"encoding/json"
	"io"

	"github.com/justonetree/pan-cli/pkg/contract"
)

func WriteOK(stdout, stderr io.Writer, meta contract.Meta, data any) int {
	return writeJSON(stdout, contract.Response{Status: "ok", Data: data, Meta: meta}, 0)
}

func WritePage(stdout, stderr io.Writer, meta contract.Meta, data any, page contract.Pagination) int {
	return writeJSON(stdout, contract.Response{Status: "ok", Data: data, Pagination: &page, Meta: meta}, 0)
}

func WriteError(stdout, stderr io.Writer, meta contract.Meta, err contract.Error) int {
	return writeJSON(stdout, contract.Response{Status: "error", Error: &err, Meta: meta}, contract.ExitCode(err.Code))
}

func writeJSON(stdout io.Writer, resp contract.Response, code int) int {
	enc := json.NewEncoder(stdout)
	enc.SetEscapeHTML(false)
	if err := enc.Encode(resp); err != nil {
		return 50
	}
	return code
}
```

`internal/output/errors.go`:

```go
package output

import "strings"

func RedactSecret(s string) string {
	if s == "" {
		return ""
	}
	replacers := []string{"UID=", "CID=", "SEID=", "KID=", "Authorization=", "Cookie="}
	out := s
	for _, marker := range replacers {
		idx := strings.Index(out, marker)
		if idx >= 0 {
			end := strings.Index(out[idx:], ";")
			if end < 0 {
				out = out[:idx+len(marker)] + "***"
			} else {
				out = out[:idx+len(marker)] + "***" + out[idx+end:]
			}
		}
	}
	return out
}
```

- [ ] **Step 5: Verify tests pass**

Run: `go test ./internal/output ./pkg/contract`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add pkg/contract internal/output
git commit -m "feat: add cli JSON contracts"
```

---

### Task 3: Config And File Credential Store

**Files:**
- Create: `internal/config/config.go`
- Create: `internal/config/config_test.go`
- Create: `internal/credential/file_store.go`
- Create: `internal/credential/file_store_test.go`

- [ ] **Step 1: Write config and credential tests**

`internal/config/config_test.go`:

```go
package config

import "testing"

func TestPathsUseProviderAndProfile(t *testing.T) {
	paths := NewPaths(t.TempDir(), "115", "default")
	if got := paths.ProfilePath(); got == "" || !hasSuffix(got, "profiles/115.default.json") {
		t.Fatalf("profile path = %q", got)
	}
	if got := paths.CredentialPath(); got == "" || !hasSuffix(got, "credentials/115.default.json") {
		t.Fatalf("credential path = %q", got)
	}
}

func hasSuffix(s, suffix string) bool {
	if len(s) < len(suffix) {
		return false
	}
	return s[len(s)-len(suffix):] == suffix
}
```

`internal/credential/file_store_test.go`:

```go
package credential

import (
	"os"
	"testing"
)

func TestFileStoreRoundTripAndPermission(t *testing.T) {
	store := NewFileStore(t.TempDir())
	secret := []byte(`{"uid":"u","cid":"c","seid":"s","kid":"k"}`)

	if err := store.Save("115", "default", secret); err != nil {
		t.Fatalf("save: %v", err)
	}
	got, err := store.Load("115", "default")
	if err != nil {
		t.Fatalf("load: %v", err)
	}
	if string(got) != string(secret) {
		t.Fatalf("secret = %s", got)
	}

	stat, err := os.Stat(store.path("115", "default"))
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if stat.Mode().Perm() != 0600 {
		t.Fatalf("mode = %v, want 0600", stat.Mode().Perm())
	}
}
```

- [ ] **Step 2: Verify tests fail**

Run: `go test ./internal/config ./internal/credential`

Expected: FAIL because packages are not implemented.

- [ ] **Step 3: Add config paths**

`internal/config/config.go`:

```go
package config

import (
	"os"
	"path/filepath"
)

type Paths struct {
	BaseDir  string
	Provider string
	Profile  string
}

func DefaultBaseDir() string {
	if dir := os.Getenv("PAN_CLI_CONFIG_DIR"); dir != "" {
		return dir
	}
	if dir, err := os.UserConfigDir(); err == nil {
		return filepath.Join(dir, "pan-cli")
	}
	return filepath.Join(".", ".pan-cli")
}

func NewPaths(baseDir, providerName, profile string) Paths {
	if profile == "" {
		profile = "default"
	}
	return Paths{BaseDir: baseDir, Provider: providerName, Profile: profile}
}

func (p Paths) ProfilePath() string {
	return filepath.Join(p.BaseDir, "profiles", p.Provider+"."+p.Profile+".json")
}

func (p Paths) CredentialPath() string {
	return filepath.Join(p.BaseDir, "credentials", p.Provider+"."+p.Profile+".json")
}
```

- [ ] **Step 4: Add credential file store**

`internal/credential/file_store.go`:

```go
package credential

import (
	"os"
	"path/filepath"
)

type FileStore struct {
	baseDir string
}

func NewFileStore(baseDir string) *FileStore {
	return &FileStore{baseDir: baseDir}
}

func (s *FileStore) Save(providerName, profile string, data []byte) error {
	dir := filepath.Join(s.baseDir, "credentials")
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	return os.WriteFile(s.path(providerName, profile), data, 0600)
}

func (s *FileStore) Load(providerName, profile string) ([]byte, error) {
	return os.ReadFile(s.path(providerName, profile))
}

func (s *FileStore) Delete(providerName, profile string) error {
	err := os.Remove(s.path(providerName, profile))
	if os.IsNotExist(err) {
		return nil
	}
	return err
}

func (s *FileStore) path(providerName, profile string) string {
	if profile == "" {
		profile = "default"
	}
	return filepath.Join(s.baseDir, "credentials", providerName+"."+profile+".json")
}
```

- [ ] **Step 5: Verify tests pass**

Run: `go test ./internal/config ./internal/credential`

Expected: PASS.

- [ ] **Step 6: Commit**

```bash
git add internal/config internal/credential
git commit -m "feat: add config and credential storage"
```

---

### Task 4: Provider Interfaces And Registry

**Files:**
- Create: `pkg/provider/provider.go`
- Create: `pkg/provider/registry.go`
- Create: `pkg/provider/registry_test.go`

- [ ] **Step 1: Write registry test**

`pkg/provider/registry_test.go`:

```go
package provider

import "testing"

type fakeProvider struct{}

func (fakeProvider) Name() string { return "fake" }
func (fakeProvider) Capabilities() Capabilities {
	return Capabilities{List: true}
}

func TestRegistryReturnsProvider(t *testing.T) {
	reg := NewRegistry()
	reg.Register(fakeProvider{})
	got, ok := reg.Get("fake")
	if !ok {
		t.Fatal("provider not found")
	}
	if got.Name() != "fake" {
		t.Fatalf("name = %q", got.Name())
	}
}
```

- [ ] **Step 2: Verify test fails**

Run: `go test ./pkg/provider`

Expected: FAIL because registry is not implemented.

- [ ] **Step 3: Implement provider interfaces**

`pkg/provider/provider.go`:

```go
package provider

type Capabilities struct {
	PathLookup    bool `json:"path_lookup"`
	List          bool `json:"list"`
	Search        bool `json:"search"`
	Mkdir         bool `json:"mkdir"`
	Rename        bool `json:"rename"`
	Move          bool `json:"move"`
	Copy          bool `json:"copy"`
	Remove        bool `json:"remove"`
	Download      bool `json:"download"`
	Upload        bool `json:"upload"`
	OfflineTask   bool `json:"offline_task"`
	Share         bool `json:"share"`
	RecycleBin    bool `json:"recycle_bin"`
	RapidUpload   bool `json:"rapid_upload"`
	CrossTransfer bool `json:"cross_transfer"`
}

type Provider interface {
	Name() string
	Capabilities() Capabilities
}
```

`pkg/provider/registry.go`:

```go
package provider

type Registry struct {
	items map[string]Provider
}

func NewRegistry() *Registry {
	return &Registry{items: map[string]Provider{}}
}

func (r *Registry) Register(p Provider) {
	r.items[p.Name()] = p
}

func (r *Registry) Get(name string) (Provider, bool) {
	p, ok := r.items[name]
	return p, ok
}
```

- [ ] **Step 4: Verify tests pass**

Run: `go test ./pkg/provider`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add pkg/provider
git commit -m "feat: add provider registry"
```

---

### Task 5: 115 Credential Model And Cookie Parsing

**Files:**
- Create: `providers/115/model/credential.go`
- Create: `providers/115/auth/cookie.go`
- Create: `providers/115/auth/cookie_test.go`

- [ ] **Step 1: Write cookie tests**

`providers/115/auth/cookie_test.go`:

```go
package auth

import "testing"

func TestParseCookieExtractsRequiredFields(t *testing.T) {
	cred, err := ParseCookie("UID=u; CID=c; SEID=s; KID=k; OTHER=x")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if cred.UID != "u" || cred.CID != "c" || cred.SEID != "s" || cred.KID != "k" {
		t.Fatalf("credential = %+v", cred)
	}
}

func TestParseCookieRejectsMissingFields(t *testing.T) {
	_, err := ParseCookie("UID=u;CID=c")
	if err == nil {
		t.Fatal("expected error")
	}
}

func TestCredentialRedaction(t *testing.T) {
	cred, err := ParseCookie("UID=abcdef;CID=c;SEID=s;KID=k")
	if err != nil {
		t.Fatalf("parse: %v", err)
	}
	if got := cred.RedactedUID(); got != "abc***def" {
		t.Fatalf("redacted uid = %q", got)
	}
}
```

- [ ] **Step 2: Verify tests fail**

Run: `go test ./providers/115/auth`

Expected: FAIL because `ParseCookie` is not implemented.

- [ ] **Step 3: Implement credential model**

`providers/115/model/credential.go`:

```go
package model

type Credential struct {
	UID  string `json:"uid"`
	CID  string `json:"cid"`
	SEID string `json:"seid"`
	KID  string `json:"kid"`
}

func (c Credential) Cookie() string {
	return "UID=" + c.UID + ";CID=" + c.CID + ";SEID=" + c.SEID + ";KID=" + c.KID
}

func (c Credential) RedactedUID() string {
	if len(c.UID) <= 6 {
		return "***"
	}
	return c.UID[:3] + "***" + c.UID[len(c.UID)-3:]
}
```

`providers/115/auth/cookie.go`:

```go
package auth

import (
	"errors"
	"net/http"
	"strings"

	model115 "github.com/justonetree/pan-cli/providers/115/model"
)

func ParseCookie(raw string) (model115.Credential, error) {
	header := http.Header{}
	header.Add("Cookie", raw)
	req := http.Request{Header: header}
	values := map[string]string{}
	for _, c := range req.Cookies() {
		values[strings.ToUpper(c.Name)] = c.Value
	}
	cred := model115.Credential{
		UID:  values["UID"],
		CID:  values["CID"],
		SEID: values["SEID"],
		KID:  values["KID"],
	}
	if cred.UID == "" || cred.CID == "" || cred.SEID == "" || cred.KID == "" {
		return model115.Credential{}, errors.New("cookie must contain UID, CID, SEID, and KID")
	}
	return cred, nil
}
```

- [ ] **Step 4: Verify tests pass**

Run: `go test ./providers/115/auth ./providers/115/model`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add providers/115/model/credential.go providers/115/auth
git commit -m "feat: parse 115 cookie credentials"
```

---

### Task 6: 115 File And Offline Model Conversion

**Files:**
- Create: `providers/115/model/file.go`
- Create: `providers/115/model/file_test.go`
- Create: `providers/115/model/offline.go`
- Create: `providers/115/model/offline_test.go`

- [ ] **Step 1: Write model conversion tests**

`providers/115/model/file_test.go`:

```go
package model

import (
	"testing"
	"time"
)

func TestFileToContract(t *testing.T) {
	updated := time.Unix(1717200000, 0)
	got := File{
		ID:        "123",
		Name:      "demo.mp4",
		IsDir:     false,
		Size:      10,
		SHA1:      "ABC",
		PickCode:  "pick",
		ThumbURL:  "thumb",
		UpdatedAt: updated,
	}.ToContract("/电影/demo.mp4")

	if got.ID != "123" || got.Name != "demo.mp4" || got.Type != "file" || got.PickCode != "pick" {
		t.Fatalf("file info = %+v", got)
	}
}
```

`providers/115/model/offline_test.go`:

```go
package model

import (
	"testing"

	"github.com/justonetree/pan-cli/pkg/contract"
)

func TestOfflineTaskToContract(t *testing.T) {
	got := OfflineTask{GID: "g", Name: "demo.mp4", StatusText: "running", Progress: 42.5}.ToContract()
	if got.GID != "g" || got.Status != contract.OfflineRunning || got.Progress != 42.5 {
		t.Fatalf("task = %+v", got)
	}
}
```

- [ ] **Step 2: Verify tests fail**

Run: `go test ./providers/115/model`

Expected: FAIL because `File` and `OfflineTask` are not implemented.

- [ ] **Step 3: Implement model conversion**

`providers/115/model/file.go`:

```go
package model

import (
	"time"

	"github.com/justonetree/pan-cli/pkg/contract"
)

type File struct {
	ID        string
	Name      string
	IsDir     bool
	Size      int64
	SHA1      string
	PickCode  string
	ThumbURL  string
	CreatedAt time.Time
	UpdatedAt time.Time
}

func (f File) ToContract(path string) contract.FileInfo {
	fileType := contract.FileTypeFile
	if f.IsDir {
		fileType = contract.FileTypeDir
	}
	var created *time.Time
	if !f.CreatedAt.IsZero() {
		created = &f.CreatedAt
	}
	var updated *time.Time
	if !f.UpdatedAt.IsZero() {
		updated = &f.UpdatedAt
	}
	return contract.FileInfo{
		ID:        f.ID,
		Name:      f.Name,
		Type:      fileType,
		Path:      path,
		Size:      f.Size,
		SHA1:      f.SHA1,
		PickCode:  f.PickCode,
		ThumbURL:  f.ThumbURL,
		CreatedAt: created,
		UpdatedAt: updated,
		Provider:  "115",
	}
}
```

`providers/115/model/offline.go`:

```go
package model

import "github.com/justonetree/pan-cli/pkg/contract"

type OfflineTask struct {
	GID        string
	Name       string
	StatusText string
	Progress   float64
	Size       int64
	FileID     string
}

func (t OfflineTask) ToContract() contract.OfflineTask {
	status := contract.OfflineUnknown
	switch t.StatusText {
	case "pending":
		status = contract.OfflinePending
	case "running":
		status = contract.OfflineRunning
	case "done", "success", "completed":
		status = contract.OfflineDone
	case "failed", "error":
		status = contract.OfflineFailed
	}
	return contract.OfflineTask{
		GID:      t.GID,
		Name:     t.Name,
		Status:   status,
		Progress: t.Progress,
		Size:     t.Size,
		FileID:   t.FileID,
	}
}
```

- [ ] **Step 4: Verify tests pass**

Run: `go test ./providers/115/model`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add providers/115/model
git commit -m "feat: add 115 model conversions"
```

---

### Task 7: 115 Provider Registration And Error Mapping

**Files:**
- Create: `providers/115/provider.go`
- Create: `providers/115/client/errors.go`
- Create: `providers/115/client/errors_test.go`

- [ ] **Step 1: Write error mapping test**

`providers/115/client/errors_test.go`:

```go
package client

import (
	"errors"
	"testing"

	"github.com/justonetree/pan-cli/pkg/contract"
)

func TestMapErrorAuthRequired(t *testing.T) {
	got := MapError(errors.New("missing cookie or qrcode account"))
	if got.Code != contract.CodeAuthRequired {
		t.Fatalf("code = %s", got.Code)
	}
}

func TestMapErrorNetwork(t *testing.T) {
	got := MapError(errors.New("dial tcp: i/o timeout"))
	if got.Code != contract.CodeNetworkError || !got.Retryable {
		t.Fatalf("error = %+v", got)
	}
}
```

- [ ] **Step 2: Verify test fails**

Run: `go test ./providers/115/client`

Expected: FAIL because `MapError` is not implemented.

- [ ] **Step 3: Implement provider capabilities and error mapping**

`providers/115/provider.go`:

```go
package pan115

import "github.com/justonetree/pan-cli/pkg/provider"

type Provider struct{}

func New() Provider {
	return Provider{}
}

func (Provider) Name() string {
	return "115"
}

func (Provider) Capabilities() provider.Capabilities {
	return provider.Capabilities{
		PathLookup:  true,
		List:        true,
		Mkdir:       true,
		Rename:      true,
		Move:        true,
		Copy:        true,
		Remove:      true,
		Download:    true,
		OfflineTask: true,
	}
}
```

`providers/115/client/errors.go`:

```go
package client

import (
	"strings"

	"github.com/justonetree/pan-cli/pkg/contract"
)

func MapError(err error) contract.Error {
	if err == nil {
		return contract.NewError(contract.CodeInternalError, "unknown error", "", false)
	}
	msg := err.Error()
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "missing cookie") || strings.Contains(lower, "qrcode"):
		return contract.NewError(contract.CodeAuthRequired, "115 authentication is required.", msg, false)
	case strings.Contains(lower, "login") && (strings.Contains(lower, "expired") || strings.Contains(lower, "check")):
		return contract.NewError(contract.CodeAuthExpired, "115 authentication expired.", msg, false)
	case strings.Contains(lower, "not exist") || strings.Contains(lower, "not found"):
		return contract.NewError(contract.CodeNotFound, "115 object was not found.", msg, false)
	case strings.Contains(lower, "timeout") || strings.Contains(lower, "dial tcp") || strings.Contains(lower, "connection"):
		return contract.NewError(contract.CodeNetworkError, "Network error while calling 115.", msg, true)
	case strings.Contains(lower, "rate") || strings.Contains(lower, "limit"):
		return contract.NewError(contract.CodeRateLimited, "115 rate limit reached.", msg, true)
	default:
		return contract.NewError(contract.CodeRemoteError, "115 returned an error.", msg, false)
	}
}
```

- [ ] **Step 4: Verify tests pass**

Run: `go test ./providers/115 ./providers/115/client`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add providers/115/provider.go providers/115/client/errors.go providers/115/client/errors_test.go
git commit -m "feat: register 115 provider"
```

---

### Task 8: 115 Client Wrapper For Login, Files, And Offline

**Files:**
- Create: `providers/115/client/client.go`
- Create: `providers/115/client/file.go`
- Create: `providers/115/client/offline.go`
- Create: `providers/115/client/appver.go`

- [ ] **Step 1: Implement client constructor and cookie login**

`providers/115/client/client.go`:

```go
package client

import (
	"context"

	driver115 "github.com/SheltonZhu/115driver/pkg/driver"
	model115 "github.com/justonetree/pan-cli/providers/115/model"
	"golang.org/x/time/rate"
)

type Client struct {
	raw     *driver115.Pan115Client
	limiter *rate.Limiter
}

func New(requestsPerSecond float64) *Client {
	c := &Client{raw: driver115.New(driver115.UA("Mozilla/5.0 115Browser/"+DefaultAppVersion))}
	if requestsPerSecond > 0 {
		c.limiter = rate.NewLimiter(rate.Limit(requestsPerSecond), 1)
	}
	return c
}

func (c *Client) Wait(ctx context.Context) error {
	if c.limiter == nil {
		return nil
	}
	return c.limiter.Wait(ctx)
}

func (c *Client) LoginCookie(ctx context.Context, cred model115.Credential) error {
	if err := c.Wait(ctx); err != nil {
		return err
	}
	rawCred := &driver115.Credential{}
	if err := rawCred.FromCookie(cred.Cookie()); err != nil {
		return err
	}
	c.raw.ImportCredential(rawCred)
	return c.raw.LoginCheck()
}
```

- [ ] **Step 2: Implement file methods**

`providers/115/client/file.go`:

```go
package client

import (
	"context"

	model115 "github.com/justonetree/pan-cli/providers/115/model"
)

type ListResult struct {
	Items    []model115.File
	Total    int
	HasMore  bool
	NextPage int
}

func (c *Client) List(ctx context.Context, dirID string, page, limit int) (ListResult, error) {
	if err := c.Wait(ctx); err != nil {
		return ListResult{}, err
	}
	return listPage(ctx, c.raw, dirID, page, limit)
}

func (c *Client) DownloadURL(ctx context.Context, pickCode, userAgent string) (string, map[string][]string, error) {
	if err := c.Wait(ctx); err != nil {
		return "", nil, err
	}
	info, err := c.raw.DownloadWithUA(pickCode, userAgent)
	if err != nil {
		return "", nil, err
	}
	return info.Url.Url, info.Header, nil
}
```

Add `listPage` by adapting `tmp/drivers/115/util.go` `getFilesPageWithThumb` into a package-private function that returns `ListResult`. Keep the query parameters from the spec and convert 115 file objects into `model115.File`.

- [ ] **Step 3: Implement mutation and offline methods**

`providers/115/client/offline.go`:

```go
package client

import (
	"context"

	model115 "github.com/justonetree/pan-cli/providers/115/model"
)

func (c *Client) OfflineList(ctx context.Context) ([]model115.OfflineTask, error) {
	if err := c.Wait(ctx); err != nil {
		return nil, err
	}
	resp, err := c.raw.ListOfflineTask(0)
	if err != nil {
		return nil, err
	}
	return convertOfflineTasks(resp.Tasks), nil
}

func (c *Client) OfflineAdd(ctx context.Context, uris []string, dstDirID string) ([]string, error) {
	if err := c.Wait(ctx); err != nil {
		return nil, err
	}
	return c.raw.AddOfflineTaskURIs(uris, dstDirID)
}

func (c *Client) OfflineDelete(ctx context.Context, hashes []string, deleteFiles bool) error {
	if err := c.Wait(ctx); err != nil {
		return err
	}
	return c.raw.DeleteOfflineTasks(hashes, deleteFiles)
}
```

Add `convertOfflineTasks` by reading the fields available on `driver115.OfflineTask` in the dependency after `go doc github.com/SheltonZhu/115driver/pkg/driver.OfflineTask`.

- [ ] **Step 4: Add app version constant**

`providers/115/client/appver.go`:

```go
package client

const DefaultAppVersion = "27.0.5.7"
```

- [ ] **Step 5: Run compilation check**

Run: `go test ./providers/115/...`

Expected: PASS after adapting dependency field names.

- [ ] **Step 6: Commit**

```bash
git add providers/115/client
git commit -m "feat: add 115 client wrapper"
```

---

### Task 9: Path Resolver

**Files:**
- Create: `internal/resolver/resolver.go`
- Create: `internal/resolver/resolver_test.go`

- [ ] **Step 1: Write resolver tests**

`internal/resolver/resolver_test.go`:

```go
package resolver

import (
	"context"
	"testing"

	"github.com/justonetree/pan-cli/pkg/contract"
)

type fakeLister struct {
	children map[string][]contract.FileInfo
}

func (f fakeLister) List(ctx context.Context, dirID string) ([]contract.FileInfo, error) {
	return f.children[dirID], nil
}

func TestResolveID(t *testing.T) {
	got, err := Resolve(context.Background(), fakeLister{}, "123")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got.ID != "123" {
		t.Fatalf("id = %q", got.ID)
	}
}

func TestResolvePath(t *testing.T) {
	l := fakeLister{children: map[string][]contract.FileInfo{
		"0":   {{ID: "1", Name: "电影", Type: contract.FileTypeDir}},
		"1":   {{ID: "2", Name: "demo.mp4", Type: contract.FileTypeFile}},
	}}
	got, err := Resolve(context.Background(), l, "/电影/demo.mp4")
	if err != nil {
		t.Fatalf("resolve: %v", err)
	}
	if got.ID != "2" {
		t.Fatalf("id = %q", got.ID)
	}
}
```

- [ ] **Step 2: Verify tests fail**

Run: `go test ./internal/resolver`

Expected: FAIL because resolver is not implemented.

- [ ] **Step 3: Implement resolver**

`internal/resolver/resolver.go`:

```go
package resolver

import (
	"context"
	"errors"
	"path"
	"regexp"
	"strings"

	"github.com/justonetree/pan-cli/pkg/contract"
)

type Lister interface {
	List(ctx context.Context, dirID string) ([]contract.FileInfo, error)
}

var numericID = regexp.MustCompile(`^\d+$`)

func Resolve(ctx context.Context, lister Lister, target string) (contract.FileInfo, error) {
	if target == "" || target == "/" {
		return contract.FileInfo{ID: "0", Name: "/", Type: contract.FileTypeDir, Path: "/", Provider: "115"}, nil
	}
	if numericID.MatchString(target) {
		return contract.FileInfo{ID: target, Provider: "115"}, nil
	}
	if !strings.HasPrefix(target, "/") {
		return contract.FileInfo{}, errors.New("target must be an id or absolute path")
	}
	current := contract.FileInfo{ID: "0", Name: "/", Type: contract.FileTypeDir, Path: "/", Provider: "115"}
	for _, part := range strings.Split(strings.Trim(path.Clean(target), "/"), "/") {
		children, err := lister.List(ctx, current.ID)
		if err != nil {
			return contract.FileInfo{}, err
		}
		found := false
		for _, child := range children {
			if child.Name == part {
				current = child
				found = true
				break
			}
		}
		if !found {
			return contract.FileInfo{}, errors.New("not found")
		}
	}
	return current, nil
}
```

- [ ] **Step 4: Verify tests pass**

Run: `go test ./internal/resolver`

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/resolver
git commit -m "feat: resolve 115 ids and paths"
```

---

### Task 10: CLI App Skeleton And Login Commands

**Files:**
- Create: `internal/app/app.go`
- Create: `internal/app/login.go`
- Modify: `cmd/pan/main.go`
- Modify: `cmd/115-cli/main.go`

- [ ] **Step 1: Implement app root and options**

`internal/app/app.go`:

```go
package app

import (
	"fmt"
	"os"
	"time"

	"github.com/spf13/cobra"
)

type Options struct {
	BinaryName      string
	DefaultProvider string
}

type Runtime struct {
	Options   Options
	JSON      bool
	Profile   string
	ConfigDir string
}

func Run(opts Options) int {
	rt := &Runtime{Options: opts, Profile: "default"}
	root := &cobra.Command{
		Use:           opts.BinaryName,
		SilenceUsage:  true,
		SilenceErrors: true,
	}
	root.PersistentFlags().BoolVar(&rt.JSON, "json", false, "write machine-readable JSON to stdout")
	root.PersistentFlags().StringVar(&rt.Profile, "profile", "default", "profile name")
	root.PersistentFlags().StringVar(&rt.ConfigDir, "config-dir", "", "config directory")
	root.AddCommand(loginCommand(rt))
	if err := root.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err.Error())
		return 1
	}
	return 0
}

func requestID() string {
	return "req_" + time.Now().UTC().Format("20060102_150405")
}
```

- [ ] **Step 2: Implement `login cookie`, `login status`, `logout`**

`internal/app/login.go`:

```go
package app

import (
	"encoding/json"
	"fmt"
	"io"
	"os"

	"github.com/justonetree/pan-cli/internal/config"
	"github.com/justonetree/pan-cli/internal/credential"
	"github.com/justonetree/pan-cli/internal/output"
	"github.com/justonetree/pan-cli/pkg/contract"
	auth115 "github.com/justonetree/pan-cli/providers/115/auth"
	client115 "github.com/justonetree/pan-cli/providers/115/client"
	"github.com/spf13/cobra"
)

func loginCommand(rt *Runtime) *cobra.Command {
	cmd := &cobra.Command{Use: "login"}
	cmd.AddCommand(loginCookieCommand(rt), loginStatusCommand(rt), logoutCommand(rt))
	return cmd
}

func loginCookieCommand(rt *Runtime) *cobra.Command {
	var fromStdin bool
	var rawCookie string
	cmd := &cobra.Command{
		Use: "cookie",
		RunE: func(cmd *cobra.Command, args []string) error {
			if fromStdin {
				b, err := io.ReadAll(os.Stdin)
				if err != nil {
					return err
				}
				rawCookie = string(b)
			}
			cred, err := auth115.ParseCookie(rawCookie)
			if err != nil {
				return err
			}
			c := client115.New(2)
			if err := c.LoginCookie(cmd.Context(), cred); err != nil {
				return err
			}
			base := rt.ConfigDir
			if base == "" {
				base = config.DefaultBaseDir()
			}
			store := credential.NewFileStore(base)
			payload, _ := json.Marshal(cred)
			if err := store.Save("115", rt.Profile, payload); err != nil {
				return err
			}
			meta := contract.Meta{Provider: "115", Profile: rt.Profile, RequestID: requestID()}
			if rt.JSON {
				code := output.WriteOK(os.Stdout, os.Stderr, meta, map[string]any{
					"authenticated": true,
					"uid":           cred.RedactedUID(),
				})
				os.Exit(code)
			}
			fmt.Fprintf(os.Stdout, "Logged in to 115 as %s\n", cred.RedactedUID())
			return nil
		},
	}
	cmd.Flags().BoolVar(&fromStdin, "stdin", false, "read cookie from stdin")
	cmd.Flags().StringVar(&rawCookie, "cookie", "", "115 cookie")
	return cmd
}

func loginStatusCommand(rt *Runtime) *cobra.Command {
	return &cobra.Command{
		Use: "status",
		RunE: func(cmd *cobra.Command, args []string) error {
			base := rt.ConfigDir
			if base == "" {
				base = config.DefaultBaseDir()
			}
			_, err := credential.NewFileStore(base).Load("115", rt.Profile)
			meta := contract.Meta{Provider: "115", Profile: rt.Profile, RequestID: requestID()}
			if rt.JSON {
				code := output.WriteOK(os.Stdout, os.Stderr, meta, map[string]any{"authenticated": err == nil})
				os.Exit(code)
			}
			if err != nil {
				fmt.Fprintln(os.Stdout, "Not logged in")
			} else {
				fmt.Fprintln(os.Stdout, "Logged in")
			}
			return nil
		},
	}
}

func logoutCommand(rt *Runtime) *cobra.Command {
	return &cobra.Command{
		Use: "logout",
		RunE: func(cmd *cobra.Command, args []string) error {
			base := rt.ConfigDir
			if base == "" {
				base = config.DefaultBaseDir()
			}
			err := credential.NewFileStore(base).Delete("115", rt.Profile)
			meta := contract.Meta{Provider: "115", Profile: rt.Profile, RequestID: requestID()}
			if rt.JSON {
				code := output.WriteOK(os.Stdout, os.Stderr, meta, map[string]any{"logged_out": err == nil})
				os.Exit(code)
			}
			fmt.Fprintln(os.Stdout, "Logged out")
			return err
		},
	}
}
```

- [ ] **Step 3: Run tests and build**

Run:

```bash
go test ./...
go build ./cmd/pan ./cmd/115-cli
```

Expected: both commands succeed.

- [ ] **Step 4: Commit**

```bash
git add internal/app cmd
git commit -m "feat: add login commands"
```

---

### Task 11: File Commands

**Files:**
- Create: `internal/app/files.go`
- Modify: `internal/app/app.go`
- Modify: `providers/115/client/file.go`

- [ ] **Step 1: Register file commands**

Modify `internal/app/app.go` so root adds file commands:

```go
root.AddCommand(loginCommand(rt))
root.AddCommand(filesCommand(rt)...)
```

- [ ] **Step 2: Implement command shell**

`internal/app/files.go`:

```go
package app

import (
	"os"

	"github.com/justonetree/pan-cli/internal/output"
	"github.com/justonetree/pan-cli/pkg/contract"
	"github.com/spf13/cobra"
)

func filesCommand(rt *Runtime) []*cobra.Command {
	return []*cobra.Command{
		{
			Use: "ls [target]",
			RunE: func(cmd *cobra.Command, args []string) error {
				target := "/"
				if len(args) > 0 {
					target = args[0]
				}
				meta := contract.Meta{Provider: "115", Profile: rt.Profile, RequestID: requestID()}
				if rt.JSON {
					code := output.WritePage(os.Stdout, os.Stderr, meta, map[string]any{
						"target": target,
						"items":  []contract.FileInfo{},
					}, contract.Pagination{Page: 1, Limit: 100, HasMore: false})
					os.Exit(code)
				}
				return nil
			},
		},
	}
}
```

- [ ] **Step 3: Replace shell with real 115 calls**

Extend `files.go` to:

1. Load saved credential with `credential.NewFileStore`.
2. Parse credential JSON into `providers/115/model.Credential`.
3. Initialize `client115.New(2)` and `LoginCookie`.
4. For `ls`, resolve target to dir ID and call `Client.List`.
5. For `download`, resolve target, call `Client.DownloadURL`.
6. For `mkdir/mv/cp/rename/rm`, resolve all inputs first, then call corresponding client methods.

Use command names:

```text
ls [target] --page --limit --all
download <target> [--output]
mkdir <parent> <name>
mv <target...> --to <dir>
cp <target...> --to <dir>
rename <target> <new-name>
rm <target...>
```

- [ ] **Step 4: Run command build**

Run:

```bash
go test ./...
go build ./cmd/pan ./cmd/115-cli
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/app/files.go providers/115/client/file.go
git commit -m "feat: add 115 file commands"
```

---

### Task 12: Offline Commands

**Files:**
- Create: `internal/app/offline.go`
- Modify: `internal/app/app.go`
- Modify: `providers/115/client/offline.go`

- [ ] **Step 1: Register offline command**

Modify `internal/app/app.go`:

```go
root.AddCommand(offlineCommand(rt))
```

- [ ] **Step 2: Implement offline commands**

`internal/app/offline.go`:

```go
package app

import (
	"os"
	"time"

	"github.com/justonetree/pan-cli/internal/output"
	"github.com/justonetree/pan-cli/pkg/contract"
	"github.com/spf13/cobra"
)

func offlineCommand(rt *Runtime) *cobra.Command {
	cmd := &cobra.Command{Use: "offline"}
	cmd.AddCommand(
		&cobra.Command{
			Use:  "list",
			RunE: runOfflineList(rt),
		},
		&cobra.Command{
			Use:  "add <url>",
			Args: cobra.ExactArgs(1),
			RunE: runOfflineAdd(rt),
		},
		&cobra.Command{
			Use:  "delete <gid>",
			Args: cobra.ExactArgs(1),
			RunE: runOfflineDelete(rt),
		},
		&cobra.Command{
			Use:  "wait <gid>",
			Args: cobra.ExactArgs(1),
			RunE: runOfflineWait(rt, 5*time.Second, 2*time.Hour),
		},
	)
	return cmd
}

func runOfflineList(rt *Runtime) func(*cobra.Command, []string) error {
	return func(cmd *cobra.Command, args []string) error {
		meta := contract.Meta{Provider: "115", Profile: rt.Profile, RequestID: requestID()}
		code := output.WriteOK(os.Stdout, os.Stderr, meta, map[string]any{"tasks": []contract.OfflineTask{}})
		os.Exit(code)
		return nil
	}
}
```

Then replace the shell body with real credential loading and `client115.Client` calls, matching the pattern from file commands.

- [ ] **Step 3: Implement wait loop**

`runOfflineWait` must:

1. Poll `OfflineList` every interval.
2. Match by `gid`.
3. Exit success when status is `done`.
4. Exit error when status is `failed`.
5. Exit `NETWORK_ERROR` when timeout expires.

- [ ] **Step 4: Verify**

Run:

```bash
go test ./...
go build ./cmd/pan ./cmd/115-cli
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/app/offline.go providers/115/client/offline.go
git commit -m "feat: add 115 offline commands"
```

---

### Task 13: QR Login Session Commands

**Files:**
- Create: `providers/115/auth/qrcode.go`
- Modify: `internal/app/login.go`

- [ ] **Step 1: Inspect 115driver QR API**

Run:

```bash
go doc github.com/SheltonZhu/115driver/pkg/driver | rg -n "QRCode|LoginApp|Session"
```

Expected: output identifies the QR session creation and login functions available in the dependency.

- [ ] **Step 2: Implement session persistence**

`providers/115/auth/qrcode.go`:

```go
package auth

import (
	"encoding/json"
	"os"
	"path/filepath"
	"time"
)

type QRSession struct {
	SessionID string    `json:"session_id"`
	Token     string    `json:"token"`
	LoginURL  string    `json:"login_url"`
	Source    string    `json:"source"`
	ExpiresAt time.Time `json:"expires_at"`
}

func SaveQRSession(dir string, s QRSession) error {
	if err := os.MkdirAll(dir, 0700); err != nil {
		return err
	}
	b, err := json.Marshal(s)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(dir, s.SessionID+".json"), b, 0600)
}

func LoadQRSession(dir, sessionID string) (QRSession, error) {
	b, err := os.ReadFile(filepath.Join(dir, sessionID+".json"))
	if err != nil {
		return QRSession{}, err
	}
	var s QRSession
	if err := json.Unmarshal(b, &s); err != nil {
		return QRSession{}, err
	}
	return s, nil
}
```

- [ ] **Step 3: Add `login qr` and `login wait`**

Modify `internal/app/login.go` so `loginCommand` includes:

```go
cmd.AddCommand(loginQRCommand(rt), loginWaitCommand(rt))
```

`login qr` returns JSON with `session_id`, `login_url`, and `expires_at`. `login wait` loads the session, calls the 115driver QR login method, saves the resulting `UID/CID/SEID/KID` credential, and outputs `{ "authenticated": true }`.

- [ ] **Step 4: Verify with build**

Run:

```bash
go test ./...
go build ./cmd/pan ./cmd/115-cli
```

Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add providers/115/auth/qrcode.go internal/app/login.go
git commit -m "feat: add 115 QR login"
```

---

### Task 14: Contract And Smoke Tests

**Files:**
- Create: `internal/app/app_test.go`
- Create: `docs/testing/115-smoke.md`

- [ ] **Step 1: Add app-level JSON smoke test**

`internal/app/app_test.go`:

```go
package app

import (
	"encoding/json"
	"os/exec"
	"testing"
)

func Test115CLIStatusJSONIsValid(t *testing.T) {
	cmd := exec.Command("go", "run", "../../cmd/115-cli", "--json", "--config-dir", t.TempDir(), "login", "status")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	var got map[string]any
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("stdout is not JSON: %s", out)
	}
	if got["status"] != "ok" {
		t.Fatalf("status = %v", got["status"])
	}
}
```

- [ ] **Step 2: Add manual smoke test guide**

`docs/testing/115-smoke.md`:

```markdown
# 115 Smoke Test

Use an isolated config directory:

```bash
export PAN_CLI_CONFIG_DIR=/tmp/pan-cli-115-smoke
```

Login:

```bash
printf '%s' "$PAN_115_COOKIE" | go run ./cmd/115-cli --json login cookie --stdin
```

Read-only checks:

```bash
go run ./cmd/115-cli --json login status
go run ./cmd/115-cli --json ls /
go run ./cmd/115-cli --json download /path/to/small-file
```

Mutation checks in a disposable directory:

```bash
go run ./cmd/115-cli --json mkdir / pan-cli-smoke
go run ./cmd/115-cli --json rename /pan-cli-smoke pan-cli-smoke-renamed
go run ./cmd/115-cli --json rm /pan-cli-smoke-renamed
```
```

- [ ] **Step 3: Verify all automated tests**

Run:

```bash
go test ./...
go build ./cmd/pan ./cmd/115-cli
```

Expected: PASS.

- [ ] **Step 4: Commit**

```bash
git add internal/app/app_test.go docs/testing/115-smoke.md
git commit -m "test: add 115 cli smoke coverage"
```

---

### Task 15: Documentation Update

**Files:**
- Modify: `README.md`
- Modify: `docs/providers/115.md`

- [ ] **Step 1: Update root README implementation status**

Add a short status table to `README.md`:

```markdown
## Implementation Status

| Provider | Phase | Status |
| --- | --- | --- |
| 115 | Phase 1 | Cookie/QR login, files, download links, offline tasks |
| 115 | Phase 2 | Upload planned |
| 115 | Phase 3 | Share-link read-only planned |
| Baidu | Planning | Provider document only |
| Aliyun | Planning | Provider document only |
```

- [ ] **Step 2: Update 115 provider doc with actual package names**

In `docs/providers/115.md`, ensure the implementation section points to:

```text
providers/115/client
providers/115/auth
providers/115/model
internal/app
internal/resolver
```

- [ ] **Step 3: Verify docs do not contain stale 115-cli directory references**

Run:

```bash
rg -n "115-cli/README|115-cli/" README.md docs
```

Expected: no references to the deleted `115-cli/README.md` path. References to the command name `115-cli` are allowed.

- [ ] **Step 4: Commit**

```bash
git add README.md docs/providers/115.md
git commit -m "docs: update 115 phase one status"
```

---

## Final Verification

Run all of these before marking the implementation complete:

```bash
go test ./...
go build ./cmd/pan ./cmd/115-cli
rg -n "TODO|TBD|panic\\(|fmt\\.Println\\(.*Cookie|SEID|KID" .
```

Expected:

1. `go test ./...` passes.
2. both binaries build.
3. search output contains no leaked secret printing and no unhandled placeholders in implementation files.

Manual verification with a real 115 account is required before release:

```bash
export PAN_CLI_CONFIG_DIR=/tmp/pan-cli-115-smoke
printf '%s' "$PAN_115_COOKIE" | go run ./cmd/115-cli --json login cookie --stdin
go run ./cmd/115-cli --json login status
go run ./cmd/115-cli --json ls /
```

Do not run mutation smoke tests against a non-disposable directory.
