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
	case strings.Contains(lower, "refresh token") || strings.Contains(lower, "access token") || strings.Contains(lower, "errno: 111") || strings.Contains(lower, "errno: -6"):
		return contract.NewError(contract.CodeAuthExpired, "Baidu authentication expired.", msg, false)
	case strings.Contains(lower, "not found") || strings.Contains(lower, "objectnotfound") || strings.Contains(lower, "errno: -9"):
		return contract.NewError(contract.CodeNotFound, "Baidu object was not found.", msg, false)
	case strings.Contains(lower, "permission") || strings.Contains(lower, "forbidden"):
		return contract.NewError(contract.CodePermissionDenied, "Baidu permission denied.", msg, false)
	case strings.Contains(lower, "timeout") || strings.Contains(lower, "dial tcp") || strings.Contains(lower, "connection"):
		return contract.NewError(contract.CodeNetworkError, "Network error while calling Baidu.", msg, true)
	case strings.Contains(lower, "rate") || strings.Contains(lower, "limit"):
		return contract.NewError(contract.CodeRateLimited, "Baidu rate limit reached.", msg, true)
	default:
		return contract.NewError(contract.CodeRemoteError, "Baidu returned an error.", msg, false)
	}
}
