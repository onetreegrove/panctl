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
