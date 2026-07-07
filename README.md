# Nacos MCP Server

Nacos 配置中心的 MCP (Model Context Protocol) Server，让 AI 助手能够直接读取和搜索 Nacos 中的配置信息。基于 [mcp-go](https://github.com/mark3labs/mcp-go) 和 [nacos-sdk-go](https://github.com/nacos-group/nacos-sdk-go) 构建，通过 stdio 与 MCP 客户端通信，不监听任何网络端口。

[![release](https://img.shields.io/github/v/release/Vinkugor/nacos-mcp?style=flat-square)](https://github.com/Vinkugor/nacos-mcp/releases)
[![license](https://img.shields.io/github/license/Vinkugor/nacos-mcp?style=flat-square)](LICENSE)

---

## 功能特性

| Tool | 说明 | 状态 |
| ---- | ---- | ---- |
| `get_config` | 根据 dataId 和 group 获取指定配置内容 | 可用 |
| `list_configs` | 列出指定 group 下的配置列表（支持分页） | 可用 |
| `search_config` | 按关键字模糊搜索配置（dataId 前缀匹配） | 可用 |
| `publish_config` | 新增/发布配置到 Nacos（支持 `dryRun` 预览） | 可用（只读模式下禁用） |
| `update_config` | 修改已有配置（支持 `dryRun` 返回当前内容与新内容对比） | 可用（只读模式下禁用） |

---

## 快速开始

### 前置要求

- 可访问的 Nacos Server（具体版本与 `NACOS_SDK_VERSION` 的对应关系见下方[环境变量](#环境变量)）
- Go 开发者使用 `go install` 时需要 Go 1.23+

### 安装

#### 方式一：npx（推荐）

在 MCP 客户端配置中添加：

```json
{
  "mcpServers": {
    "nacos": {
      "command": "npx",
      "args": ["-y", "@vinkugor/nacos-mcp"],
      "env": {
        "NACOS_HOST": "localhost:8848",
        "NACOS_NAMESPACE": "your-namespace",
        "NACOS_USERNAME": "nacos",
        "NACOS_PASSWORD": "nacos"
      }
    }
  }
}
```

首次启动时 npx 会自动下载对应平台的二进制并缓存。

#### 方式二：go install（Go 开发者）

```bash
go install github.com/Vinkugor/nacos-mcp@latest
```

然后在 MCP 客户端配置中将 `command` 指向安装的二进制（确保 `$GOBIN` 或 `$GOPATH/bin` 在 PATH 中）：

```json
{
  "mcpServers": {
    "nacos": {
      "command": "nacos-mcp",
      "env": {
        "NACOS_HOST": "localhost:8848",
        "NACOS_NAMESPACE": "your-namespace",
        "NACOS_USERNAME": "nacos",
        "NACOS_PASSWORD": "nacos"
      }
    }
  }
}
```

#### 方式三：预编译二进制（离线 / 内网）

从 [Releases](https://github.com/Vinkugor/nacos-mcp/releases) 下载对应平台的压缩包，解压后将 `command` 指向二进制绝对路径：

```json
{
  "mcpServers": {
    "nacos": {
      "command": "/usr/local/bin/nacos-mcp",
      "env": {
        "NACOS_HOST": "localhost:8848",
        "NACOS_NAMESPACE": "your-namespace",
        "NACOS_USERNAME": "nacos",
        "NACOS_PASSWORD": "nacos"
      }
    }
  }
}
```

---

## 环境变量

### 必填配置

| 变量 | 说明 | 示例 |
| ---- | ---- | ---- |
| `NACOS_HOST` | Nacos 服务地址 | `localhost:8848` |
| `NACOS_USERNAME` | 登录用户名 | `nacos` |
| `NACOS_PASSWORD` | 登录密码 | `nacos` |

### 可选配置

| 变量 | 说明 | 默认值 |
| ---- | ---- | ------ |
| `NACOS_NAMESPACE` | 命名空间名称或 UUID（留空为 public） | 空 |
| `NACOS_CONTEXT_PATH` | API 路径前缀（自定义部署时修改） | `/nacos` |
| `NACOS_SDK_VERSION` | SDK 版本（`v2` 适配 Nacos Server v2/v3，`v1` 适配 v1） | `v2` |
| `NACOS_READONLY` | 只读模式（设为 `true` 时禁用发布和修改功能） | `false` |

### 配置说明

**命名空间识别**  
`NACOS_NAMESPACE` 可以填写命名空间的显示名称（如 `dev-env`）或 UUID（如 `cff9f83d-...`）。服务启动时会自动将显示名称转换为 UUID，转换失败时会输出警告但不影响启动。

**安全建议**  
生产环境请修改默认用户名和密码。使用默认凭证启动时会输出安全警告。

**实时性保证**  
本服务禁用了本地快照缓存，所有配置读取都直连 Nacos Server，确保获取最新配置而非陈旧缓存。

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

## 技术栈

| 组件 | 版本 |
| ---- | ---- |
| Go | 1.23+ |
| mcp-go | v0.7.0 |
| nacos-sdk-go (v2) | v2.3.5 |
| nacos-sdk-go (v1) | v1.1.6 |

## License

[Apache License 2.0](LICENSE)