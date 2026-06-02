package contract

type ErrorCode string

const (
	CodeUsageError            ErrorCode = "USAGE_ERROR"
	CodeAuthRequired          ErrorCode = "AUTH_REQUIRED"
	CodeAuthExpired           ErrorCode = "AUTH_EXPIRED"
	CodePermissionDenied      ErrorCode = "PERMISSION_DENIED"
	CodeNotFound              ErrorCode = "NOT_FOUND"
	CodeConflict              ErrorCode = "CONFLICT"
	CodeUnsupportedCapability ErrorCode = "UNSUPPORTED_CAPABILITY"
	CodeRateLimited           ErrorCode = "RATE_LIMITED"
	CodeNetworkError          ErrorCode = "NETWORK_ERROR"
	CodeRemoteError           ErrorCode = "REMOTE_ERROR"
	CodeInternalError         ErrorCode = "INTERNAL_ERROR"
)

type Error struct {
	Code      ErrorCode `json:"code"`
	Message   string    `json:"message"`
	Detail    string    `json:"detail,omitempty"`
	Retryable bool      `json:"retryable"`
}

func NewError(code ErrorCode, message, detail string, retryable bool) Error {
	return Error{Code: code, Message: message, Detail: detail, Retryable: retryable}
}

func ExitCode(code ErrorCode) int {
	switch code {
	case CodeUsageError:
		return 2
	case CodeAuthRequired:
		return 10
	case CodeAuthExpired:
		return 11
	case CodePermissionDenied:
		return 12
	case CodeNotFound:
		return 20
	case CodeConflict:
		return 21
	case CodeUnsupportedCapability:
		return 22
	case CodeRateLimited:
		return 30
	case CodeNetworkError:
		return 31
	case CodeRemoteError:
		return 40
	default:
		return 50
	}
}
