# Provider 能力契约

Provider 能力用于告诉 CLI、脚本和 Agent 某个网盘支持哪些操作。公共命令在执行前应检查 capability，不支持时返回 `UNSUPPORTED_CAPABILITY`。

## 基础能力

| capability | 说明 |
| --- | --- |
| `path_lookup` | 支持通过 `/path` 查找文件或目录 |
| `list` | 支持列目录 |
| `search` | 支持搜索 |
| `mkdir` | 支持创建目录 |
| `rename` | 支持重命名 |
| `move` | 支持移动 |
| `copy` | 支持复制 |
| `remove` | 支持删除 |
| `download` | 支持下载或获取下载链接 |
| `upload` | 支持上传 |

## 扩展能力

| capability | 说明 |
| --- | --- |
| `offline_task` | 支持离线下载任务 |
| `share` | 支持分享链接 |
| `recycle_bin` | 支持回收站 |
| `rapid_upload` | 支持秒传或极速上传 |
| `cross_transfer` | 支持跨 provider 传输优化 |

## 示例

```json
{
  "provider": "115",
  "capabilities": {
    "path_lookup": true,
    "list": true,
    "search": true,
    "mkdir": true,
    "rename": true,
    "move": true,
    "copy": true,
    "remove": true,
    "download": true,
    "upload": false,
    "offline_task": true,
    "share": false,
    "recycle_bin": false,
    "rapid_upload": false,
    "cross_transfer": false
  }
}
```

能力声明必须保守。没有实现并测试的能力不得标记为 `true`。
