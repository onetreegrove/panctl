# JSON 输出契约

本文档定义 `panctl`、`115-cli`、`baidu-cli`、`aliyun-cli` 和所有 provider 特殊命令的机器输出格式。

## 基本规则

1. 携带 `--json` 时，`stdout` 只能输出一个完整 JSON 对象。
2. 调试日志、网络细节、进度条和提示信息必须输出到 `stderr`。
3. 成功命令返回退出码 `0`。
4. 失败命令返回非 `0` 退出码，同时在 `stdout` 输出错误 JSON。
5. JSON 字段名使用 snake_case。
6. 敏感字段不得出现在 JSON 中，包括 Cookie、Token、Refresh Token、Authorization header。

## 成功响应

```json
{
  "status": "ok",
  "data": {},
  "meta": {
    "provider": "115",
    "profile": "default",
    "request_id": "req_20260601_000001"
  }
}
```

`data` 的结构由具体命令定义，但必须保持稳定。字段新增必须向后兼容，字段删除或类型变化需要 major version。

## 分页响应

分页命令必须包含 `pagination`：

```json
{
  "status": "ok",
  "data": {
    "items": []
  },
  "pagination": {
    "page": 1,
    "limit": 100,
    "total": 1234,
    "has_more": true,
    "next_page": 2
  },
  "meta": {
    "provider": "baidu",
    "profile": "default",
    "request_id": "req_20260601_000002"
  }
}
```

如果 provider 无法提供准确 total，`total` 可省略，但必须提供 `has_more`。

## 错误响应

```json
{
  "status": "error",
  "error": {
    "code": "AUTH_REQUIRED",
    "message": "Current profile is not authenticated.",
    "detail": "Run `panctl login 115` to authenticate this profile.",
    "retryable": false
  },
  "meta": {
    "provider": "115",
    "profile": "default",
    "request_id": "req_20260601_000003"
  }
}
```

`message` 面向人类，`code` 面向脚本和 Agent。自动化逻辑只能依赖 `code`、`retryable` 和退出码，不依赖 `message` 文案。

## 文件对象

公共文件命令返回的文件对象应尽量使用统一字段：

```json
{
  "id": "123456789",
  "name": "demo.mp4",
  "type": "file",
  "path": "/电影/demo.mp4",
  "size": 1048576,
  "created_at": "2026-06-01T12:00:00+08:00",
  "updated_at": "2026-06-01T12:00:00+08:00",
  "provider": "115",
  "raw": {}
}
```

`raw` 用于保留 provider 原始字段，默认不展示敏感内容。公共逻辑不得依赖 `raw`。
