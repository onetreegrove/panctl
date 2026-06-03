package client

import (
	"errors"
	"testing"

	"github.com/onetreegrove/panctl/pkg/contract"
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
