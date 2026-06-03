# 百度网盘 Provider 实现方案

百度网盘 provider 接入 `panctl` 的公共契约。当前实现以 `tmp/drivers/baidu_netdisk` 为主参考，优先支持主账号文件系统能力；分享链接读取参考 `tmp/drivers/baidu_share`，后续作为特殊命令单独接入。

## 入口

```bash
panctl ls baidu:/资料
baidu-cli ls /资料
```

`baidu-cli <cmd>` 等价于 `panctl --provider baidu <cmd>`。当前 `panctl` 主入口仍以 115 为默认 provider，跨 provider 参数解析后续统一补齐。

## 认证

第一版使用百度开放平台 refresh token：

```bash
baidu-cli login refresh-token --token <refresh-token>
```

可选参数：

```bash
--client-id <client-id>
--client-secret <client-secret>
--skip-check
```

默认 `client_id/client_secret` 沿用 `tmp/drivers/baidu_netdisk` 中的公开客户端配置。凭证保存到 `internal/credential` 管理的 `baidu.<profile>.json`，输出只展示脱敏 refresh token。

## 当前能力声明

| capability | 状态 | 说明 |
| --- | --- | --- |
| `path_lookup` | true | 通过路径逐级 list 解析 |
| `list` | true | `/xpan/file method=list` |
| `search` | false | `tmp/drivers/baidu_netdisk` 未提供搜索实现 |
| `mkdir` | true | `/xpan/file method=create isdir=1` |
| `rename` | true | `/xpan/file method=filemanager opera=rename` |
| `move` | true | `/xpan/file method=filemanager opera=move` |
| `copy` | true | `/xpan/file method=filemanager opera=copy` |
| `remove` | true | `/xpan/file method=filemanager opera=delete` |
| `download` | true | 官方 dlink，返回直链和 `User-Agent` |
| `upload` | true | precreate + superfile2 分片上传 |
| `offline_task` | false | 未发现稳定参考实现 |
| `share` | false | 分享链接读取先不混入主账号 provider |
| `recycle_bin` | false | 暂不实现 |
| `rapid_upload` | true | `create` + 单块 MD5 秒传 |
| `cross_transfer` | false | 暂不实现 |

能力声明必须保守。没有实现并通过本地契约测试或人工 smoke 的能力不得标记为 `true`。

## 公共命令映射

| panctl 命令 | 百度 API 参考 |
| --- | --- |
| `login refresh-token` | `https://openapi.baidu.com/oauth/2.0/token` |
| `login status` | 本地凭证存在性检查 |
| `logout` | 删除本地凭证 |
| `ls` | `/xpan/file?method=list` |
| `download` | `/xpan/multimedia?method=filemetas&dlink=1` |
| `mkdir` | `/xpan/file?method=create` |
| `mv` | `/xpan/file?method=filemanager&opera=move` |
| `cp` | `/xpan/file?method=filemanager&opera=copy` |
| `rename` | `/xpan/file?method=filemanager&opera=rename` |
| `rm` | `/xpan/file?method=filemanager&opera=delete` |
| `upload` | `/xpan/file?method=precreate` + `superfile2` + `create` |

## 分享链接后续方案

`tmp/drivers/baidu_share` 支持只读分享访问：

```bash
baidu-cli share-ls <share-url-or-surl> [dir]
baidu-cli share-download <share-url-or-surl> <file-id>
```

该路线依赖 `surl/pwd/BDUSS`，只提供 list 和 download，不支持文件变动或上传。它应作为 provider 特殊命令实现，不参与 `panctl ls baidu:/path` 的主账号文件树。

## 暂不实现

1. 搜索。
2. 离线下载任务。
3. 回收站。
4. 分享创建。
5. 跨 provider 传输优化。
6. Cookie/青春版登录路线。

## 测试要求

1. 本地单元测试覆盖 credential 脱敏、能力声明、文件模型转换、错误映射、MD5 helper。
2. CLI smoke 覆盖 `baidu-cli --json login status` 和 `login refresh-token --skip-check`。
3. 真实账号 smoke 需要人工提供 refresh token 后执行 `ls/download/mkdir/mv/cp/rename/rm/upload`。
