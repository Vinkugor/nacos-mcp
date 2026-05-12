# Nacos MCP Server

Nacos 配置中心的 MCP (Model Context Protocol) Server，让 AI 助手能够直接读取和搜索 Nacos 中的配置信息。基于 [mcp-go](https://github.com/mark3labs/mcp-go) 和 [nacos-sdk-go](https://github.com/nacos-group/nacos-sdk-go) 构建（同时支持 v1 与 v2，兼容 Nacos Server 1.x / 2.x），通过 stdio 方式与 MCP 客户端通信。

[![release](https://img.shields.io/github/v/release/Vinkugor/nacos-mcp?style=flat-square)](https://github.com/Vinkugor/nacos-mcp/releases)
[![license](https://img.shields.io/github/license/Vinkugor/nacos-mcp?style=flat-square)](LICENSE)

---

## 功能特性

| Tool | 说明 | 状态 |
| ---- | ---- | ---- |
| `get_config` | 根据 dataId 和 group 获取指定配置内容 | 可用 |
| `list_configs` | 列出指定 group 下的配置列表（支持分页） | 可用 |
| `search_config` | 按关键字模糊搜索配置（dataId 前缀匹配） | 可用 |
| `publish_config` | 新增/发布配置到 Nacos（支持 `dryRun` 预览；只读模式下不注册） | 可用 |
| `update_config` | 修改已有配置（支持 `dryRun` 返回当前内容与新内容对比；只读模式下不注册） | 可用 |

---

## 快速开始

### 前置要求

- 可访问的 Nacos Server：2.x（默认，走 gRPC）或 1.x（需设置 `NACOS_SDK_VERSION=v1`，走 HTTP）
- 源码构建时需 Go 1.23+

### 安装

#### 方式一：npx（推荐）

```bash
npx @vinkugor/nacos-mcp@latest
```

或在 MCP 客户端配置：

```json
{
  "mcpServers": {
    "nacos": {
      "command": "npx",
      "args": ["-y", "@vinkugor/nacos-mcp"],
      "env": {
        "NACOS_HOST": "localhost:8848",
        "NACOS_NAMESPACE": "your-namespace-id",
        "NACOS_USERNAME": "nacos",
        "NACOS_PASSWORD": "nacos"
      }
    }
  }
}
```

#### 方式二：下载预编译二进制

到 [Releases](https://github.com/Vinkugor/nacos-mcp/releases) 页面下载对应平台的压缩包。一键脚本示例（macOS / Linux）：

```bash
VERSION=$(curl -s https://api.github.com/repos/Vinkugor/nacos-mcp/releases/latest | grep tag_name | cut -d '"' -f 4 | sed 's/^v//')
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m | sed 's/x86_64/amd64/;s/aarch64/arm64/')
curl -L "https://github.com/Vinkugor/nacos-mcp/releases/download/v${VERSION}/nacos-mcp_${VERSION}_${OS}_${ARCH}.tar.gz" | tar xz
sudo mv nacos-mcp /usr/local/bin/
```

#### 方式三：go install

```bash
go install github.com/Vinkugor/nacos-mcp@latest
```

#### 方式四：源码构建

```bash
git clone https://github.com/Vinkugor/nacos-mcp.git
cd nacos-mcp
make build           # 当前平台
make build-all       # 全平台（Linux/macOS/Windows）
```

### 运行

```bash
NACOS_HOST=localhost:8848 nacos-mcp
```

服务启动后通过 stdio 接收 MCP 协议请求，不会监听网络端口。

---

## 环境变量

| 变量 | 说明 | 默认值 |
| ---- | ---- | ------ |
| `NACOS_HOST` | Nacos 服务地址（host:port） | `localhost:8848` |
| `NACOS_NAMESPACE` | Nacos 命名空间 ID | 空（public） |
| `NACOS_USERNAME` | Nacos 用户名 | `nacos` |
| `NACOS_PASSWORD` | Nacos 密码 | `nacos` |
| `NACOS_SDK_VERSION` | 使用的 nacos-sdk-go 主版本。`v2` 走 gRPC，要求 Nacos Server ≥ 2.x；`v1` 走 HTTP，兼容 Nacos Server 1.x | `v2` |
| `NACOS_READONLY` | 只读模式：为 `true` 时不注册 `publish_config` 和 `update_config` 工具，禁止发布与修改配置 | `false` |

> 生产环境请务必修改默认用户名/密码。使用默认凭证启动时会在 stderr 输出 WARNING。
>
> 本服务**禁用**了 Nacos SDK 的本地快照（snapshot/failover cache），所有读操作都直连 Nacos Server，避免在服务端不可达时返回陈旧配置。

---

## IDE / Claude Desktop 配置

在 MCP 客户端的配置文件中添加以下内容：

```json
{
  "mcpServers": {
    "nacos": {
      "command": "/usr/local/bin/nacos-mcp",
      "env": {
        "NACOS_HOST": "localhost:8848",
        "NACOS_NAMESPACE": "your-namespace-id",
        "NACOS_USERNAME": "nacos",
        "NACOS_PASSWORD": "nacos",
        "NACOS_SDK_VERSION": "v2"
      }
    }
  }
}
```

---

## Makefile 命令

| 命令 | 说明 |
| ---- | ---- |
| `make build` | 编译当前平台二进制 |
| `make build-all` | 编译全平台二进制（Linux/macOS/Windows） |
| `make run` | `go run .` 启动服务 |
| `make test` | 运行测试 |
| `make deps` | 下载并整理依赖 |
| `make clean` | 清理构建产物 |
| `make install` | 编译并安装到 `/usr/local/bin/` |

---

## 发布流程

项目使用 [GoReleaser](https://goreleaser.com/) + GitHub Actions 自动发布多平台二进制。

```bash
git tag v0.1.0
git push origin v0.1.0
```

推送 `v*` tag 后会自动触发 `.github/workflows/release.yml`，在 Releases 页生成 Linux/macOS/Windows 的 amd64/arm64 压缩包及 `checksums.txt`。

本地预览（不发布）：

```bash
brew install goreleaser
goreleaser release --snapshot --clean --skip=publish
```

---

## 技术栈

| 组件 | 版本 |
| ---- | ---- |
| Go | 1.23+ |
| mcp-go | v0.7.0 |
| nacos-sdk-go (v2) | v2.3.5 |
| nacos-sdk-go (v1) | v1.1.6 |

## License

[Apache License 2.0](LICENSE)