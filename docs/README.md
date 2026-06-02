# pan-cli - 多网盘命令行平台设计方案

`pan-cli` 是一个面向人类用户、脚本和 AI Agent 的多网盘 CLI 平台。项目先实现 115 网盘，后续扩展百度网盘、阿里云盘等 provider。核心原则是：**统一 CLI 契约、统一凭证与输出基础设施、provider 独立适配具体网盘能力**。

---

## 设计目标

1. 提供统一入口 `pan`，支持跨网盘操作和统一 target 语法。
2. 保留 provider 专属入口，例如 `115-cli`、`baidu-cli`、`aliyun-cli`，降低单网盘用户的使用成本。
3. 所有入口共享同一套 Go core、JSON 输出契约、错误码、配置、凭证和 npm 安装逻辑。
4. provider 只实现具体网盘 API、认证流程和差异化能力，避免复制 CLI 框架。
5. AI Skill 只依赖稳定机器契约，不依赖人类输出文案。

---

## 推荐仓库结构

```text
pan/
├── README.md
├── go.work
├── go.mod
├── go.sum
├── Makefile
├── package.json
│
├── docs/
│   ├── contracts/
│   │   ├── json-output.md
│   │   ├── errors.md
│   │   └── provider-capabilities.md
│   └── providers/
│       ├── 115.md
│       ├── baidu.md
│       └── aliyun.md
│
├── cmd/
│   ├── pan/                       # 统一入口：pan ls 115:/path
│   ├── 115-cli/                   # 兼容入口：115-cli ls /path
│   ├── baidu-cli/                 # 兼容入口：baidu-cli ls /path
│   └── aliyun-cli/                # 兼容入口：aliyun-cli ls /path
│
├── internal/
│   ├── app/                       # CLI 装配、命令注册、入口模式切换
│   ├── config/                    # profile、配置目录、权限、迁移
│   ├── credential/                # Keychain/Credential Manager/Secret Service/file fallback
│   ├── output/                    # JSON、人类表格、错误输出
│   ├── resolver/                  # pan target 解析：115:/a、baidu:/b
│   ├── transfer/                  # 上传、下载、进度、断点续传通用能力
│   └── release/                   # version、build metadata
│
├── pkg/
│   ├── contract/                  # 稳定对外类型：FileInfo、Error、Pagination
│   └── provider/                  # Provider interface 与能力声明
│
├── providers/
│   ├── 115/
│   │   ├── auth/
│   │   ├── client/
│   │   ├── commands/
│   │   └── provider.go
│   ├── baidu/
│   │   ├── auth/
│   │   ├── client/
│   │   ├── commands/
│   │   └── provider.go
│   └── aliyun/
│       ├── auth/
│       ├── client/
│       ├── commands/
│       └── provider.go
│
├── npm/
│   ├── pan-cli/
│   ├── 115-cli/
│   ├── baidu-cli/
│   └── aliyun-cli/
│
├── scripts/
│   ├── run.js
│   ├── install.js
│   └── install-wizard.js
│
└── skills/
    ├── pan-cli/
    ├── 115-cli/
    ├── baidu-cli/
    └── aliyun-cli/
```

---

## 入口设计

### 统一入口

`pan` 是完整能力入口，target 必须显式带 provider：

```bash
pan ls 115:/电影
pan ls baidu:/资料
pan download aliyun:/backup/a.zip --output ./a.zip
pan cp 115:/a.zip aliyun:/backup/
```

统一入口适合跨网盘迁移、聚合搜索、AI Agent 编排和自动化脚本。

### Provider 专属入口

专属入口只是在启动时注入默认 provider，不复制业务逻辑：

```bash
115-cli ls /电影
baidu-cli ls /资料
aliyun-cli ls /backup
```

等价关系：

```bash
115-cli ls /电影
pan --provider 115 ls /电影
pan ls 115:/电影
```

---

## 公共命令与 Provider 特殊命令

公共命令只覆盖跨网盘最小文件系统能力：

| 命令 | 说明 |
| --- | --- |
| `login` / `logout` / `login status` | 认证生命周期 |
| `ls` | 列目录 |
| `mkdir` | 创建目录 |
| `rm` | 删除 |
| `mv` | 移动 |
| `cp` | 复制 |
| `rename` | 重命名 |
| `download` | 下载或生成下载链接 |
| `upload` | 上传 |
| `search` | 搜索 |
| `info` | 账号、容量、版本信息 |

Provider 特殊命令放在 provider 自己的命名空间或专属入口下：

```bash
115-cli offline add <url>
115-cli offline wait <gid>
baidu-cli rapid-upload <file>
aliyun-cli share create <target>
```

不要为了统一而把 `offline`、`rapid-upload`、`share`、`秒传` 等能力塞进公共接口。公共层通过 capability 判断某个 provider 是否支持某项能力。

---

## Provider 抽象

Provider 接口只描述最小稳定边界：

```go
type Provider interface {
    Name() string
    Capabilities() Capabilities
    Auth() AuthService
    Files() FileService
}

type Capabilities struct {
    Upload        bool
    Download      bool
    OfflineTask   bool
    Share         bool
    RecycleBin    bool
    PathLookup    bool
    CrossTransfer bool
}
```

公共命令只依赖 `AuthService`、`FileService` 和 capability。Provider 特殊命令可以直接依赖 provider 内部 client，但输出仍必须遵守通用 JSON 与错误契约。

---

## 配置与凭证目录

统一使用 `~/.config/pan-cli`，不要为每个网盘建立互不兼容的配置体系：

```text
~/.config/pan-cli/
├── config.json
├── profiles/
│   ├── 115.default.json
│   ├── baidu.default.json
│   └── aliyun.default.json
└── credentials/
    ├── 115.default.json
    ├── baidu.default.json
    └── aliyun.default.json
```

凭证优先写入系统安全存储：

1. macOS Keychain。
2. Windows Credential Manager。
3. Linux Secret Service。
4. 不可用时回退到 `credentials/<provider>.<profile>.json`，目录权限 `0700`，文件权限 `0600`。

日志、JSON、错误 detail 和 panic 输出都不得包含 Cookie、Token、Refresh Token 原文。

---

## 输出与错误契约

所有入口和 provider 共享同一套机器契约：

1. `--json` 下 stdout 只能输出一个完整 JSON 对象。
2. 调试日志、进度、网络细节写入 stderr。
3. 成功响应包含 `status=data/meta`。
4. 分页响应包含 `pagination`。
5. 错误响应包含稳定 `error.code` 和非 0 退出码。

详见：

- [JSON 输出契约](docs/contracts/json-output.md)
- [错误码契约](docs/contracts/errors.md)
- [Provider 能力契约](docs/contracts/provider-capabilities.md)

---

## npm 分发策略

发布多个 npm 包，但它们共享同一套二进制和安装脚本：

| npm 包 | bin | 默认行为 |
| --- | --- | --- |
| `pan-cli` | `pan` | 统一入口 |
| `115-cli` | `115-cli` | 默认 provider 为 `115` |
| `baidu-cli` | `baidu-cli` | 默认 provider 为 `baidu` |
| `aliyun-cli` | `aliyun-cli` | 默认 provider 为 `aliyun` |

npm 包装层只负责：

1. 检查当前平台二进制是否存在。
2. 下载并校验 Release artifact 的 SHA256。
3. 透传 stdin/stdout/stderr、参数、信号和退出码。

---

## Implementation Status

| Provider | Phase | Status |
| --- | --- | --- |
| 115 | Phase 1 | Cookie/QR login, files, download links, offline tasks |
| 115 | Phase 2 | Upload planned |
| 115 | Phase 3 | Share-link read-only planned |
| Baidu | Planning | Provider document only |
| Aliyun | Planning | Provider document only |

---

## 开发顺序

1. 建立根级 `pan-cli` monorepo 结构和通用契约文档。
2. 实现 `pan` 入口、公共输出、错误码、配置、凭证和 provider registry。
3. 将 115 作为第一个 provider 落地，实现登录、`ls`、`download`、`search`。
4. 补齐 115 变更类命令和离线任务特殊命令。
5. 增加 `115-cli` npm 包和 Skill。
6. 按同一 provider interface 增加百度网盘。
7. 增加阿里云盘。
8. 实现跨 provider 的复制、迁移、聚合搜索等高级能力。
