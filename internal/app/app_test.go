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
