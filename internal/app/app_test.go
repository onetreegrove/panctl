package app

import (
	"context"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
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

func TestBaiduCLIStatusJSONIsValid(t *testing.T) {
	cmd := exec.Command("go", "run", "../../cmd/baidu-cli", "--json", "--config-dir", t.TempDir(), "login", "status")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	var got struct {
		Status string `json:"status"`
		Meta   struct {
			Provider string `json:"provider"`
		} `json:"meta"`
		Data struct {
			Authenticated bool `json:"authenticated"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("stdout is not JSON: %s", out)
	}
	if got.Status != "ok" || got.Meta.Provider != "baidu" || got.Data.Authenticated {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestBaiduCLIRefreshTokenLoginPersistsCredential(t *testing.T) {
	configDir := t.TempDir()
	cmd := exec.Command("go", "run", "../../cmd/baidu-cli", "--json", "--config-dir", configDir, "login", "refresh-token", "--token", "refresh-token-value", "--skip-check")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("command failed: %v; out=%s", err, out)
	}
	var got struct {
		Status string `json:"status"`
		Meta   struct {
			Provider string `json:"provider"`
		} `json:"meta"`
		Data struct {
			Authenticated bool   `json:"authenticated"`
			RefreshToken  string `json:"refresh_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("stdout is not JSON: %s", out)
	}
	if got.Status != "ok" || got.Meta.Provider != "baidu" || !got.Data.Authenticated {
		t.Fatalf("unexpected response: %+v", got)
	}
	if got.Data.RefreshToken != "refr***alue" {
		t.Fatalf("refresh token was not redacted: %+v", got.Data)
	}
}

func TestAliyunCLIStatusJSONIsValid(t *testing.T) {
	cmd := exec.Command("go", "run", "../../cmd/aliyun-cli", "--json", "--config-dir", t.TempDir(), "login", "status")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("command failed: %v", err)
	}
	var got struct {
		Status string `json:"status"`
		Meta   struct {
			Provider string `json:"provider"`
		} `json:"meta"`
		Data struct {
			Authenticated bool `json:"authenticated"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("stdout is not JSON: %s", out)
	}
	if got.Status != "ok" || got.Meta.Provider != "aliyun" || got.Data.Authenticated {
		t.Fatalf("unexpected response: %+v", got)
	}
}

func TestAliyunCLIRefreshTokenLoginPersistsCredential(t *testing.T) {
	configDir := t.TempDir()
	cmd := exec.Command("go", "run", "../../cmd/aliyun-cli", "--json", "--config-dir", configDir, "login", "refresh-token", "--token", "refresh-token-value", "--skip-check")
	out, err := cmd.Output()
	if err != nil {
		t.Fatalf("command failed: %v; out=%s", err, out)
	}
	var got struct {
		Status string `json:"status"`
		Meta   struct {
			Provider string `json:"provider"`
		} `json:"meta"`
		Data struct {
			Authenticated bool   `json:"authenticated"`
			RefreshToken  string `json:"refresh_token"`
		} `json:"data"`
	}
	if err := json.Unmarshal(out, &got); err != nil {
		t.Fatalf("stdout is not JSON: %s", out)
	}
	if got.Status != "ok" || got.Meta.Provider != "aliyun" || !got.Data.Authenticated {
		t.Fatalf("unexpected response: %+v", got)
	}
	if got.Data.RefreshToken != "refr***alue" {
		t.Fatalf("refresh token was not redacted: %+v", got.Data)
	}
}

func TestGetProviderOpsSupportsAliyun(t *testing.T) {
	configDir := t.TempDir()
	providerDir := filepath.Join(configDir, "credentials")
	if err := os.MkdirAll(providerDir, 0700); err != nil {
		t.Fatal(err)
	}
	cred := map[string]any{
		"refresh_token": "refresh-token-value",
		"access_token":  "access-token-value",
		"drive_id":      "drive_1",
		"user_id":       "user_1",
		"drive_type":    "resource",
	}
	b, err := json.Marshal(cred)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(providerDir, "aliyun.default.json"), b, 0600); err != nil {
		t.Fatal(err)
	}
	rt := &Runtime{Options: Options{DefaultProvider: "aliyun"}, Profile: "default", ConfigDir: configDir}
	ops, meta, err := getProviderOps(rt, context.Background())
	if err != nil {
		t.Fatalf("getProviderOps returned error: %v", err)
	}
	if ops == nil || meta.Provider != "aliyun" {
		t.Fatalf("unexpected ops/meta: %T %+v", ops, meta)
	}
}
