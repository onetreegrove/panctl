package output

import (
	"encoding/json"
	"io"

	"github.com/onetreegrove/panctl/pkg/contract"
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
