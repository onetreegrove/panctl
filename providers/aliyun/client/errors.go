package client

import (
	"strings"

	"github.com/justonetree/pan-cli/pkg/contract"
)

func MapError(err error) contract.Error {
	if err == nil {
		return contract.NewError(contract.CodeInternalError, "unknown error", "", false)
	}
	msg := err.Error()
	lower := strings.ToLower(msg)
	switch {
	case strings.Contains(lower, "accesstokeninvalid") || strings.Contains(lower, "accesstokenexpired") || strings.Contains(lower, "i400jd") || strings.Contains(lower, "refresh token"):
		return contract.NewError(contract.CodeAuthExpired, "Aliyun authentication expired.", msg, false)
	case strings.Contains(lower, "filenotfound") || strings.Contains(lower, "not found"):
		return contract.NewError(contract.CodeNotFound, "Aliyun object was not found.", msg, false)
	case strings.Contains(lower, "forbidden") || strings.Contains(lower, "permission"):
		return contract.NewError(contract.CodePermissionDenied, "Aliyun permission denied.", msg, false)
	case strings.Contains(lower, "429") || strings.Contains(lower, "rate") || strings.Contains(lower, "limit") || strings.Contains(lower, "too many"):
		return contract.NewError(contract.CodeRateLimited, "Aliyun rate limit reached.", msg, true)
	case strings.Contains(lower, "timeout") || strings.Contains(lower, "dial tcp") || strings.Contains(lower, "connection"):
		return contract.NewError(contract.CodeNetworkError, "Network error while calling Aliyun.", msg, true)
	default:
		return contract.NewError(contract.CodeRemoteError, "Aliyun returned an error.", msg, false)
	}
}
