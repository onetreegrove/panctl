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
		code contract.ErrorCode
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
