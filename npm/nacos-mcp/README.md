# @vinkugor/nacos-mcp

Nacos 配置中心的 MCP (Model Context Protocol) Server，让 AI 助手能够直接读取和搜索 Nacos 中的配置信息。

## 安装

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

## 支持平台

- Linux x64 / arm64
- macOS x64 / arm64 (Apple Silicon)
- Windows x64 / arm64

不支持的平台会提示从 [Releases](https://github.com/Vinkugor/nacos-mcp/releases) 手动下载。

## 环境变量

| 变量 | 说明 | 默认值 |
| ---- | ---- | ------ |
| `NACOS_HOST` | Nacos 服务地址（host:port） | `localhost:8848` |
| `NACOS_NAMESPACE` | Nacos 命名空间 ID | 空（public） |
| `NACOS_USERNAME` | Nacos 用户名 | `nacos` |
| `NACOS_PASSWORD` | Nacos 密码 | `nacos` |
| `NACOS_READONLY` | 只读模式：禁用写操作工具 | `false` |

更多信息见 [主仓库](https://github.com/Vinkugor/nacos-mcp)。

## License

Apache-2.0
