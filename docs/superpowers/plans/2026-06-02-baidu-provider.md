# Baidu Provider Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a Baidu Netdisk provider that follows the existing `pan-cli` provider contract and mirrors the common 115 file commands where Baidu supports them.

**Architecture:** Use `tmp/drivers/baidu_netdisk` as the primary reference: refresh-token credentials, XPan list/filemanager/download APIs, and precreate/superfile2 upload. Keep Baidu API code under `providers/baidu`, keep provider selection in `internal/app`, and expose `baidu-cli` as a provider-specific entrypoint.

**Tech Stack:** Go, Cobra, Resty, existing `internal/credential`, `internal/output`, `internal/resolver`, and `pkg/contract`.

---

### Task 1: Provider Contract And Credential Tests

**Files:**
- Create: `providers/baidu/provider.go`
- Create: `providers/baidu/model/credential.go`
- Create: `providers/baidu/model/file.go`
- Create: `providers/baidu/client/errors.go`
- Test: `providers/baidu/model/credential_test.go`
- Test: `providers/baidu/model/file_test.go`
- Test: `providers/baidu/provider_test.go`
- Test: `providers/baidu/client/errors_test.go`

- [ ] Write failing tests for conservative Baidu capabilities, refresh-token credential redaction, file conversion, and error mapping.
- [ ] Run the tests and confirm they fail because Baidu packages do not exist.
- [ ] Implement the provider, credential model, file model, and error mapper.
- [ ] Run package tests and confirm they pass.

### Task 2: Baidu Client

**Files:**
- Create: `providers/baidu/client/client.go`
- Create: `providers/baidu/client/file.go`
- Create: `providers/baidu/client/upload.go`
- Create: `providers/baidu/client/hash.go`
- Test: `providers/baidu/client/hash_test.go`

- [ ] Write failing tests for MD5 helpers and request-independent behavior.
- [ ] Implement refresh-token API wrappers, list, download link, mkdir, move, copy, rename, delete, rapid upload, and multipart upload based on `tmp/drivers/baidu_netdisk`.
- [ ] Keep network behavior untested locally unless a fake HTTP server is added.

### Task 3: CLI Provider Selection

**Files:**
- Create: `cmd/baidu-cli/main.go`
- Modify: `internal/app/app.go`
- Modify: `internal/app/login.go`
- Modify: `internal/app/files.go`
- Modify: `internal/app/offline.go`
- Test: `internal/app/app_test.go`

- [ ] Write failing tests for `baidu-cli --json login status` and `baidu-cli --json login refresh-token --token`.
- [ ] Add runtime provider selection helpers.
- [ ] Route common file commands to either 115 or Baidu.
- [ ] Return `UNSUPPORTED_CAPABILITY` for Baidu offline commands.

### Task 4: Documentation And Verification

**Files:**
- Modify: `docs/providers/baidu.md`

- [ ] Rewrite the Baidu provider doc with implemented and deferred capabilities.
- [ ] Run `go test ./...`.
- [ ] Run `go build ./cmd/pan ./cmd/115-cli ./cmd/baidu-cli` if sandbox permissions allow.
