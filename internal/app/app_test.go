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
