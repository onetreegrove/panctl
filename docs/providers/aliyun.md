# 阿里云盘 Provider 实现方案

阿里云盘 provider 接入 `pan-cli` 的公共契约。当前方案以 `tmp/drivers/aliyundrive_open` 为主参考，优先支持主账号文件系统能力；分享链接读取参考 `tmp/drivers/aliyundrive_share`，后续作为 provider 特殊命令单独接入。

`tmp/drivers/aliyundrive` 是非开放接口路线，依赖设备签名和 `device_id`，上游已经标注 deprecated。第一阶段不采用该路线，避免把不稳定认证流程引入公共 provider。

## 入口

```bash
pan ls aliyun:/备份
aliyun-cli ls /备份
aliyun-cli login refresh-token --token <refresh-token>
```

`aliyun-cli <cmd>` 等价于 `pan --provider aliyun <cmd>`。当前 `pan` 主入口仍以 115 为默认 provider，跨 provider 参数解析后续统一补齐。

## 认证

第一版使用阿里云盘开放接口 refresh token：

```bash
aliyun-cli login refresh-token --token <refresh-token>
```

可选参数：

```bash
--client-id <client-id>
--client-secret <client-secret>
--oauth-token-url <url>
--drive-type default|resource|backup
--skip-check
```

认证流程参考 `tmp/drivers/aliyundrive_open/util.go`：

1. 使用 refresh token 换取 access token。
2. 调用 `/adrive/v1.0/user/getDriveInfo` 获取用户信息和 drive id。
3. 按 `drive_type` 选择 `default_drive_id`、`resource_drive_id` 或 `backup_drive_id`，默认使用 `resource`。
4. access token 失效时自动刷新，并保存新的 refresh token。

凭证保存到 `internal/credential` 管理的 `aliyun.<profile>.json`，输出只展示脱敏 refresh token。provider 不允许自行写明文 token 文件。

## 当前能力声明

| capability | 状态 | 说明 |
| --- | --- |
| `path_lookup` | true | 通过路径逐级 list 解析 |
| `list` | true | `/adrive/v1.0/openFile/list` |
| `search` | false | `tmp/drivers/aliyundrive_open` 未提供搜索实现 |
| `mkdir` | true | `/adrive/v1.0/openFile/create type=folder` |
| `rename` | true | `/adrive/v1.0/openFile/update` |
| `move` | true | `/adrive/v1.0/openFile/move` |
| `copy` | true | `/adrive/v1.0/openFile/copy` |
| `remove` | true | 默认进入回收站，可配置永久删除 |
| `download` | true | `/adrive/v1.0/openFile/getDownloadUrl` |
| `upload` | true | `create` + 分片 PUT + `complete` |
| `offline_task` | false | 未发现稳定参考实现 |
| `share` | false | 分享链接读取先不混入主账号 provider |
| `recycle_bin` | false | 仅支持删除到回收站，不提供回收站列表/恢复能力 |
| `rapid_upload` | true | `pre_hash` + SHA1 + `proof_code` 秒传 |
| `cross_transfer` | false | 暂不实现 |

能力声明必须保守。没有实现并通过本地契约测试或人工 smoke 的能力不得标记为 `true`。

## 公共命令映射

| pan-cli 命令 | 阿里云盘 OpenAPI 参考 |
| --- | --- |
| `login refresh-token` | `/oauth/access_token` 或自定义 `oauth_token_url` |
| `login status` | 本地凭证存在性检查，可选调用 `/adrive/v1.0/user/getDriveInfo` |
| `logout` | 删除本地凭证 |
| `ls` | `/adrive/v1.0/openFile/list` |
| `download` | `/adrive/v1.0/openFile/getDownloadUrl` |
| `mkdir` | `/adrive/v1.0/openFile/create` |
| `mv` | `/adrive/v1.0/openFile/move` |
| `cp` | `/adrive/v1.0/openFile/copy` |
| `rename` | `/adrive/v1.0/openFile/update` |
| `rm` | `/adrive/v1.0/openFile/recyclebin/trash` 或 `/adrive/v1.0/openFile/delete` |
| `upload` | `/adrive/v1.0/openFile/create` + `PUT upload_url` + `/adrive/v1.0/openFile/complete` |

## 文件模型

阿里云盘文件对象需要转换为公共 `contract.FileInfo`：

```json
{
  "id": "file_id",
  "name": "demo.mp4",
  "type": "file",
  "path": "/备份/demo.mp4",
  "size": 1048576,
  "sha1": "abcdef...",
  "thumb_url": "https://...",
  "created_at": "2026-06-02T12:00:00+08:00",
  "updated_at": "2026-06-02T12:00:00+08:00",
  "provider": "aliyun",
  "raw": {}
}
```

目录的 `type` 为 `dir`，`id` 使用阿里云盘 `file_id`。根目录默认 ID 为 `root`，路径解析由 `internal/resolver` 通过 `ListChildren` 逐级完成。

可保留到 `raw` 的字段：

| 字段 | 说明 |
| --- | --- |
| `drive_id` | 当前 drive id |
| `parent_file_id` | 父目录 file id |
| `category` | 阿里云盘分类 |
| `content_hash` | 文件 SHA1 |

## 上传设计

上传参考 `tmp/drivers/aliyundrive_open/upload.go`，流程如下：

1. 根据文件大小计算分片大小，默认 20 MiB；大文件需要调大分片，避免超过 10000 个分片。
2. 调用 `/adrive/v1.0/openFile/create` 创建上传任务，传入 `part_info_list`。
3. 如果启用秒传且文件大于 100 KiB，先上传前 1024 字节 SHA1 作为 `pre_hash`。
4. 服务端返回 `PreHashMatched` 时，计算完整 SHA1 和 `proof_code` 后再次调用 `create`。
5. 如果 `rapid_upload=true`，直接完成；否则逐片 `PUT upload_url`。
6. 上传 URL 超过 50 分钟时，通过 `/adrive/v1.0/openFile/getUploadUrl` 重新获取。
7. 调用 `/adrive/v1.0/openFile/complete` 完成上传，并返回新文件模型。

`proof_code` 的计算依赖 access token：取 access token MD5 前 16 位转换为整数，对文件大小取模后读取最多 8 字节，再做 base64。该逻辑必须单元测试覆盖，避免秒传路径 silently 失效。

## 下载设计

下载链接通过 `/adrive/v1.0/openFile/getDownloadUrl` 获取：

```json
{
  "drive_id": "<drive_id>",
  "file_id": "<file_id>",
  "expire_sec": 14400
}
```

返回 JSON 时需要输出 URL、必要 headers 和过期时间。上游 OpenAPI 代码没有可靠解析实际过期时间，第一版可以按请求的 `expire_sec` 计算保守 `expires_at`；如果 API 返回空 URL，`.livp` 文件需要按配置读取 `streamsUrl.jpeg` 或 `streamsUrl.mov`。

## 请求限速

阿里云盘 OpenAPI 限速按用户和应用维度生效。第一版参考 `tmp/drivers/aliyundrive_open/limiter.go`，按用户共享 limiter：

| 请求类型 | 建议速率 |
| --- | --- |
| list | 3.9 req/s |
| link | 0.9 req/s |
| other | 14.9 req/s |

token refresh 前无法知道 user id，可先使用全局 limiter；拿到 user id 后切换到用户级 limiter。限速等待必须响应 context cancellation。

## 错误映射

阿里云盘错误需要映射到公共错误码：

| 阿里云盘错误 | 公共错误码 |
| --- | --- |
| `AccessTokenInvalid`、`AccessTokenExpired`、`I400JD` | `AUTH_EXPIRED`，刷新 token 后重试一次 |
| `FileNotFound`、`NotFound` | `NOT_FOUND` |
| `Forbidden`、权限不足 | `PERMISSION_DENIED` |
| HTTP 429、限流提示 | `RATE_LIMITED` |
| 网络超时、连接错误 | `NETWORK_ERROR` |
| 其他远端错误 | `REMOTE_ERROR` |

刷新 token 后必须保存新 refresh token；如果新旧 JWT `sub` 不一致，应视为认证失败，避免串号。

## 分享链接后续方案

`tmp/drivers/aliyundrive_share` 支持只读分享访问：

```bash
aliyun-cli share-ls <share-url-or-id> [dir]
aliyun-cli share-download <share-url-or-id> <file-id>
```

该路线依赖 `share_id/share_pwd/share_token`，可列分享目录并获取下载链接，不支持文件变动或上传。它应作为 provider 特殊命令实现，不参与 `pan ls aliyun:/path` 的主账号文件树。

## 暂不实现

1. 搜索。
2. 离线下载任务。
3. 回收站列表、恢复、清空。
4. 分享创建。
5. 分享链接写入能力。
6. 非开放接口 `aliyundrive` 路线。
7. 跨 provider 传输优化。

## 测试要求

1. 本地单元测试覆盖 credential 脱敏、能力声明、文件模型转换、错误映射、part size 计算、proof code 计算。
2. CLI smoke 覆盖 `aliyun-cli --json login status` 和 `login refresh-token --skip-check`。
3. 真实账号 smoke 需要人工提供 refresh token 后执行 `ls/download/mkdir/mv/cp/rename/rm/upload`。
4. 上传 smoke 需要分别覆盖普通分片上传和秒传命中路径。

## 接入要求

1. 遵守 [JSON 输出契约](../contracts/json-output.md)。
2. 遵守 [错误码契约](../contracts/errors.md)。
3. 遵守 [Provider 能力契约](../contracts/provider-capabilities.md)。
4. 凭证必须由 `internal/credential` 管理，不允许 provider 自行写明文 token 文件。
5. 公共命令执行前必须检查 capability，不支持时返回 `UNSUPPORTED_CAPABILITY`。
6. provider 能力声明必须与已实现、已测试的行为一致。
