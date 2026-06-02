# 错误码契约

所有入口和 provider 必须使用统一错误码。Provider 原始错误需要映射到本表。

| code | 退出码 | retryable | 说明 |
| --- | ---: | --- | --- |
| `USAGE_ERROR` | 2 | false | 参数错误或命令用法错误 |
| `AUTH_REQUIRED` | 10 | false | 未登录或凭证不存在 |
| `AUTH_EXPIRED` | 11 | false | 凭证过期，需要重新登录 |
| `PERMISSION_DENIED` | 12 | false | 权限不足或账号受限 |
| `NOT_FOUND` | 20 | false | 文件、目录或任务不存在 |
| `CONFLICT` | 21 | false | 目标重名、重复任务等冲突 |
| `UNSUPPORTED_CAPABILITY` | 22 | false | Provider 不支持该能力 |
| `RATE_LIMITED` | 30 | true | 被 provider 限流 |
| `NETWORK_ERROR` | 31 | true | DNS、连接、超时等网络错误 |
| `REMOTE_ERROR` | 40 | false | Provider API 返回未分类业务错误 |
| `INTERNAL_ERROR` | 50 | false | CLI 内部错误 |

## 映射原则

1. 认证类错误优先映射到 `AUTH_REQUIRED` 或 `AUTH_EXPIRED`。
2. 请求可重试时设置 `retryable=true`，Agent 默认最多重试三次。
3. Provider 不支持的公共命令必须返回 `UNSUPPORTED_CAPABILITY`，而不是 `REMOTE_ERROR`。
4. 原始 provider 错误可以进入 `error.detail`，但必须脱敏。
5. 非 JSON 模式下错误摘要写入 `stderr`，退出码保持一致。
