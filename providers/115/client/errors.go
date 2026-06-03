package client

import (
	"strings"

	"github.com/onetreegrove/panctl/pkg/contract"
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
