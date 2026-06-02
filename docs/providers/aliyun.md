# 阿里云盘 Provider 规划

阿里云盘 provider 后续接入 `pan-cli` 的公共契约。本文档先定义边界，具体 API 细节在实现前补充。

## 入口

```bash
pan ls aliyun:/备份
aliyun-cli ls /备份
```

`aliyun-cli <cmd>` 等价于 `pan --provider aliyun <cmd>`。

## 预期公共能力

| 能力 | 规划 |
| --- | --- |
| 登录 | 支持扫码、refresh token 或其他稳定认证方式 |
| 列目录 | 支持 |
| 搜索 | 支持 |
| 下载 | 支持 |
| 上传 | 规划支持 |
| 创建/删除/移动/重命名 | 规划支持 |

## 可能的特殊能力

```bash
aliyun-cli share create <target>
aliyun-cli share list
```

分享等 provider 差异较大的能力先放在 provider 特殊命令中，不进入公共接口。

## 接入要求

1. 遵守 [JSON 输出契约](../contracts/json-output.md)。
2. 遵守 [错误码契约](../contracts/errors.md)。
3. 凭证必须由 `internal/credential` 管理，不允许 provider 自行写明文 token 文件。
