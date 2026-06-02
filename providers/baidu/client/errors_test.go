package client

import (
	"errors"
	"testing"

	"github.com/justonetree/pan-cli/pkg/contract"
)

func TestMapErrorAuthExpired(t *testing.T) {
	got := MapError(errors.New("errno: 111 access token expired"))

	if got.Code != contract.CodeAuthExpired {
		t.Fatalf("code = %s", got.Code)
	}
}

func TestMapErrorNotFound(t *testing.T) {
	got := MapError(errors.New("object not found"))

	if got.Code != contract.CodeNotFound {
		t.Fatalf("code = %s", got.Code)
	}
}
