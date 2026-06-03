[![License: MIT](https://img.shields.io/badge/License-MIT-yellow.svg)](https://opensource.org/licenses/MIT)
[![Go Version](https://img.shields.io/badge/go-%3E%3D1.22-blue.svg)](https://go.dev/)

[中文版](./README.md)

`panctl` 是一个面向人类用户、自动化脚本和 AI Agent 的多网盘统一命令行平台。首期已完整支持 115 网盘，后续逐步扩展百度网盘、阿里云盘等供应商。核心原则是：**统一 CLI 契约、统一凭证与输出基础设施、各网盘 Provider 独立适配**。

[安装与快速开始](#安装与快速开始) · [双层命令调用](#双层命令调用) · [认证设计](#认证设计) · [安全与防泄露](#安全与风险提示使用前必读) · [贡献](#贡献)

---

## 为什么选择 panctl？

- **为 Agent 原生设计** — 支持全局 `--json` 标志，输出结构化且契约稳定的单行 JSON，错误输出经过脱敏，AI Agent 可以无缝、安全地进行自动化调度。
- **功能完备 (Phase 1)** — 首期已支持 Cookie/扫码登录、状态查询、绝对路径/ID 级别解析、常规文件系统变动（ls/mkdir/mv/cp/rename/rm）、安全下载及离线下载任务控制。
- **原子性下载保护** — 采用临时文件写入 + 成功后原子重命名的机制，防止中途网络中断或异常退出 clobber（损坏）本地已有的目标文件。
- **三层架构设计** — 共享的核心凭证存储（Keychain/OS 密钥链与本地 0600 文件回退）、路径解析层，以及插拔式的网盘 Provider 层，保持架构极度清爽。
- **双入口调用** — 支持统一入口 `panctl` 跨网盘操作，同时支持专属入口 `115-cli`，直接绑定默认 provider 降低使用成本。

---

## 功能矩阵

| 类别 | 模块能力 | 命令行参数例 | 说明 |
| :--- | :--- | :--- | :--- |
| 🔐 **凭证认证** | Cookie 导入、扫码登录、状态查询、登出 | `login cookie`, `login qr`, `status` | 支持 OS 密钥链及 0600 文件回退，防止凭证泄漏。 |
| 📂 **常规文件** | 列目录、分页检索、全量加载、元数据获取 | `ls [target]`, `ls / --all` | 支持绝对路径与 115 ID 两种 target。 |
| 🛠️ **文件变动** | 新建文件夹、重命名、移动、复制、删除 | `mkdir`, `rename`, `mv`, `cp`, `rm` | 支持多 target 统一解析与原子批处理，不进行无效的 API 调用。 |
| ⚡ **安全下载** | 下载 URL 提取、Header 获取、原子断点下载 | `download <target> --output` | 未指定 output 时仅输出下载 URL 与 UA headers，指定后安全写入。 |
| 📥 **离线任务** | 离线任务添加、列表查询、删除、等待轮询 | `offline add`, `offline wait <gid>` | 原生适配 115 离线磁力与 HTTP 任务，支持轮询与超时监控。 |

---

## 安装与快速开始

### 环境要求
* Go 1.22+（编译安装）
* 运行环境：Mac/Linux/Windows 终端

### 1. 编译安装
```bash
git clone https://github.com/onetreegrove/panctl.git
cd panctl

# 编译出多网盘主入口 (panctl) 以及 115 专属入口 (115-cli)
go build ./cmd/panctl ./cmd/115-cli
```

### 2. 账号登录（人类用户）
* **方式一：Cookie 登录**
  ```bash
  # 交互式粘贴或使用 stdin 安全输入
  echo "UID=xxx; CID=xxx; SEID=xxx; KID=xxx" | ./115-cli login cookie --stdin
  ```
* **方式二：扫码登录 (QR)**
  ```bash
  # 1. 启动扫码会话，生成 115 App 登录链接
  ./115-cli login qr
  
  # 2. 手机 115 App 扫码同意后，用输出的 session_id 确认并登录
  ./115-cli login wait <session_id>
  ```

### 3. 开始使用
```bash
# 查询登录状态
./115-cli login status

# 列出根目录下的文件
./115-cli ls /
```

---

## 快速开始（AI Agent）

如果您是 AI 编码助手或 Agent，请按如下流程操作：

1. **调用构建**: 运行 `go test ./... && go build ./cmd/panctl ./cmd/115-cli` 确保本地环境可用。
2. **凭证检查**: 运行 `./115-cli --json login status` 获取登录状态。若未登录，则读取环境变量中的 Cookie 或引导人类用户扫码。
3. **调用参数**: 始终在命令后追加 `--json` 参数，获取易于解析的机器响应对象。例如：
   ```bash
   ./115-cli --json ls /电影
   ```

---

## 双层命令调用

### 1. 统一入口 `panctl`
适用于跨网盘聚合操作，target 必须显式指定 provider 命名空间：
```bash
./panctl ls 115:/电影
./panctl cp 115:/a.zip baidu:/backup/  # 跨网盘迁移（规划中）
```

### 2. Provider 专属入口 `115-cli`
在启动时自动注入默认的 provider 为 `115`，省去路径前缀输入：
```bash
./115-cli ls /电影
```

---

## 进阶与输出格式

可以通过 `--json` 让 CLI 输入输出完全走 JSON 通信。

### 成功响应格式
```json
{
  "status": "ok",
  "data": {
    "items": [
      {
        "id": "123456789",
        "name": "demo.mp4",
        "type": "file",
        "size": 1048576,
        "sha1": "ABCDEF...",
        "pick_code": "ecjq9i...",
        "provider": "115"
      }
    ]
  },
  "meta": {
    "provider": "115",
    "profile": "default",
    "request_id": "req_20260602_172054"
  }
}
```

### 异常响应与错误码
错误响应中包含公共错误码、脱敏的错误信息以及是否可重试标志：
```json
{
  "status": "error",
  "error": {
    "code": "AUTH_EXPIRED",
    "message": "115 authentication expired.",
    "detail": "sso check failed",
    "retryable": false
  },
  "meta": {
    "provider": "115",
    "profile": "default",
    "request_id": "req_20260602_172054"
  }
}
```

关于核心退出码映射请查阅 [错误码契约](docs/contracts/errors.md)。

---

## 安全与风险提示（使用前必读）

1. **凭证脱敏保障**：在全局错误输出、日志流和 panic trace 中，涉及 `UID=`、`CID=`、`SEID=`、`KID=` 等任何敏感鉴权字段，均会在写入控制台或管道前被过滤脱敏为 `***`，彻底规避 AI Agent 将敏感凭证录入 context 或对外泄露的风险。
2. **私有存储权限**：如果系统安全密钥链（OS Keychain）不可用，本地回退保存的凭证目录权限强制为 `0700`，文件权限强制为 `0600`，仅当前操作系统用户可读写。
3. **谨慎对外暴露**：本工具建议仅作为私人终端或 Agent 执行沙箱内的助手工具。请勿将此 CLI 暴露于公开无鉴权的 Web 接口或群聊环境中，以免导致网盘数据泄露或被恶意删除。

---

## 许可证

本项目基于 **MIT 许可证** 开源。
本软件调用 115 网盘开放或底层 API，使用这些 API 请遵守相关平台的服务协议和隐私政策。
