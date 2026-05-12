package main

import (
	"fmt"
	"log"
	"os"
	"strconv"

	"github.com/mark3labs/mcp-go/mcp"
	"github.com/mark3labs/mcp-go/server"
)

var version = "dev"

const (
	defaultNacosUsername = "nacos"
	defaultNacosPassword = "nacos"
)

func main() {
	if len(os.Args) == 2 && (os.Args[1] == "--version" || os.Args[1] == "-v") {
		fmt.Println(version)
		os.Exit(0)
	}

	config := &NacosConfig{
		ServerAddr: getEnv("NACOS_HOST", "localhost:8848"),
		Namespace:  getEnv("NACOS_NAMESPACE", ""),
		Username:   getEnv("NACOS_USERNAME", defaultNacosUsername),
		Password:   getEnv("NACOS_PASSWORD", defaultNacosPassword),
		ReadOnly:   getEnvBool("NACOS_READONLY", false),
		SDKVersion: SDKVersion(getEnv("NACOS_SDK_VERSION", string(SDKVersionV2))),
	}

	if config.Username == defaultNacosUsername && config.Password == defaultNacosPassword {
		log.Printf("WARNING: using default Nacos credentials (nacos/nacos). Set NACOS_USERNAME and NACOS_PASSWORD for production use.")
	}

	nacosClient, err := NewNacosClient(config)
	if err != nil {
		log.Fatalf("Failed to initialize Nacos client: %v", err)
	}
	defer nacosClient.Close()

	s := server.NewMCPServer(
		"nacos-mcp-server",
		version,
	)

	registerTools(s, nacosClient)

	log.Printf("Nacos MCP server %s running on stdio", version)
	log.Printf("Connected to: %s", config.ServerAddr)
	log.Printf("Namespace: %s", config.Namespace)
	log.Printf("Read-only mode: %v", config.ReadOnly)
	log.Printf("Nacos SDK version: %s", config.SDKVersion)

	if err := server.ServeStdio(s); err != nil {
		log.Fatalf("Server error: %v", err)
	}
}

func registerTools(s *server.MCPServer, client NacosClient) {
	s.AddTool(mcp.Tool{
		Name: "get_config",
		Description: "" +
			"读取 Nacos 配置中心中一条指定配置的完整内容。\n" +
			"Read a single Nacos configuration by its dataId and group.\n\n" +
			"Use this tool when you know the exact dataId and need the full config content.\n" +
			"If you don't know the dataId, first use search_config (fuzzy match) or list_configs (browse by group).\n\n" +
			"The returned content is the raw config text — parse it as YAML, JSON, Properties, TOML, or plain text depending on the config's type.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"dataId": map[string]interface{}{
					"type":        "string",
					"description": "配置唯一标识符。通常以文件扩展名暗示内容类型，如 application.yaml、datasource.json、nacos.properties。\nConfiguration data ID. Naming convention: the file extension indicates the config type (e.g., application.yaml, db.properties, config.json).",
				},
				"group": map[string]interface{}{
					"type":        "string",
					"description": "配置所属分组，用于隔离不同环境/应用。\nConfiguration group used to isolate configs by environment or application.",
					"default":     "DEFAULT_GROUP",
				},
			},
			Required: []string{"dataId"},
		},
	}, func(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
		dataId, _ := arguments["dataId"].(string)
		group, ok := arguments["group"].(string)
		if !ok || group == "" {
			group = "DEFAULT_GROUP"
		}

		content, err := client.GetConfig(dataId, group)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error: %v", err)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Configuration: %s (group: %s)\n\n%s", dataId, group, content)), nil
	})

	s.AddTool(mcp.Tool{
		Name: "list_configs",
		Description: "" +
			"列出指定 group 下所有配置的 dataId（不含配置内容）。\n" +
			"List all configuration dataIds in a group — returns names only, not the content.\n\n" +
			"Use this tool when you need to browse or discover what configurations exist in a group.\n" +
			"Supports pagination via pageNo/pageSize (default page 1, 100 items per page).\n\n" +
			"Tip: after you find the dataId you need, call get_config to read its actual content.\n" +
			"If you are looking for a specific keyword, prefer search_config which does fuzzy matching across groups.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"group": map[string]interface{}{
					"type":        "string",
					"description": "配置分组名称。\nTarget configuration group to browse.",
					"default":     "DEFAULT_GROUP",
				},
				"pageNo": map[string]interface{}{
					"type":        "number",
					"description": "页码，从 1 开始。\nPage number, starts at 1.",
					"default":     1,
				},
				"pageSize": map[string]interface{}{
					"type":        "number",
					"description": "每页返回条数。\nItems per page.",
					"default":     100,
				},
			},
		},
	}, func(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
		group, ok := arguments["group"].(string)
		if !ok || group == "" {
			group = "DEFAULT_GROUP"
		}

		pageNo := 1
		if pn, ok := arguments["pageNo"].(float64); ok {
			pageNo = int(pn)
		}

		pageSize := 100
		if ps, ok := arguments["pageSize"].(float64); ok {
			pageSize = int(ps)
		}

		page, err := client.ListConfigs(group, pageNo, pageSize)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error: %v", err)), nil
		}

		if len(page.PageItems) == 0 {
			return mcp.NewToolResultText(fmt.Sprintf("No configurations found in group %s", group)), nil
		}

		result := fmt.Sprintf("Found %d configuration(s) in group %s (total: %d):\n\n", len(page.PageItems), group, page.TotalCount)
		for _, cfg := range page.PageItems {
			g := cfg.Group
			if g == "" {
				g = group
			}
			result += fmt.Sprintf("- %s (group: %s)\n", cfg.DataId, g)
		}

		return mcp.NewToolResultText(result), nil
	})

	s.AddTool(mcp.Tool{
		Name: "search_config",
		Description: "" +
			"按关键字模糊搜索 dataId，返回匹配的配置列表（不含内容）。\n" +
			"Fuzzy-search configuration dataIds by keyword — returns matching names, not the content.\n\n" +
			"Use this tool when you don't know the exact dataId and need to find it by a partial keyword (e.g. 'mysql', 'redis').\n" +
			"The search does prefix matching on dataId fields.\n" +
			"Optionally narrow by group to limit results to a specific group.\n\n" +
			"Tip: after finding matching configs, use get_config with the returned dataId to read the full content.\n" +
			"If you need to see ALL configs in a group (not keyword-filtered), use list_configs instead.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"search": map[string]interface{}{
					"type":        "string",
					"description": "搜索关键字，对 dataId 做前缀匹配。\nSearch keyword for prefix-matching against dataId (e.g., 'mysql', 'redis', 'application').",
				},
				"group": map[string]interface{}{
					"type":        "string",
					"description": "可选，限定搜索的分组。不填则跨所有分组搜索。\nOptional group filter. Leave empty to search across all groups.",
				},
			},
			Required: []string{"search"},
		},
	}, func(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
		search, _ := arguments["search"].(string)
		group, _ := arguments["group"].(string)

		page, err := client.SearchConfig(search, group)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error: %v", err)), nil
		}

		if len(page.PageItems) == 0 {
			groupText := ""
			if group != "" {
				groupText = fmt.Sprintf(" in group %s", group)
			}
			return mcp.NewToolResultText(fmt.Sprintf("No configurations found matching \"%s\"%s", search, groupText)), nil
		}

		result := fmt.Sprintf("Found %d configuration(s) matching \"%s\":\n\n", len(page.PageItems), search)
		for _, cfg := range page.PageItems {
			g := cfg.Group
			if g == "" {
				g = group
			}
			if g != "" {
				result += fmt.Sprintf("- %s (group: %s)\n", cfg.DataId, g)
			} else {
				result += fmt.Sprintf("- %s\n", cfg.DataId)
			}
		}

		return mcp.NewToolResultText(result), nil
	})

	if client.IsReadOnly() {
		log.Printf("Read-only mode enabled, skipping registration of publish_config and update_config tools")
		return
	}

	s.AddTool(mcp.Tool{
		Name: "publish_config",
		Description: "" +
			"创建新配置到 Nacos。\n" +
			"Create a NEW configuration in Nacos.\n\n" +
			"IMPORTANT: This tool will FAIL if the dataId already exists. To modify an existing config, use update_config instead.\n\n" +
			"configType must be one of: yaml, json, properties, toml, text.\n" +
			"The dataId filename extension should match the configType (e.g., mysql.yaml → yaml, db.properties → properties).\n\n" +
			"Set dryRun=true to preview the content that would be published without actually writing — always recommended before the final publish.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"dataId": map[string]interface{}{
					"type":        "string",
					"description": "新配置的唯一标识符，不能与已有 dataId 重复。\nUnique ID for the new config. Must not already exist in the target group.",
				},
				"group": map[string]interface{}{
					"type":        "string",
					"description": "配置所属分组。\nTarget group for the new config.",
					"default":     "DEFAULT_GROUP",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "配置内容文本。\nFull configuration content as text.",
				},
				"configType": map[string]interface{}{
					"type":        "string",
					"description": "配置类型，必须是以下之一：yaml, json, properties, toml, text。\nConfig type, must be one of: yaml, json, properties, toml, text.",
					"default":     "yaml",
				},
				"dryRun": map[string]interface{}{
					"type":        "boolean",
					"description": "设为 true 时只预览将要发布的 payload，不实际写入。建议发布前先用此模式确认。\nSet true to preview without writing. Always recommended before final publish.",
					"default":     false,
				},
			},
			Required: []string{"dataId", "content"},
		},
	}, func(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
		dataId, _ := arguments["dataId"].(string)
		group, ok := arguments["group"].(string)
		if !ok || group == "" {
			group = "DEFAULT_GROUP"
		}
		content, _ := arguments["content"].(string)
		configType, ok := arguments["configType"].(string)
		if !ok || configType == "" {
			configType = "yaml"
		}
		dryRun, _ := arguments["dryRun"].(bool)

		if dryRun {
			return mcp.NewToolResultText(fmt.Sprintf(
				"[dry-run] Would publish config: %s (group: %s, type: %s)\n\n--- Content to publish ---\n%s",
				dataId, group, configType, content,
			)), nil
		}

		ok2, err := client.PublishConfig(dataId, group, content, configType)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error: %v", err)), nil
		}

		if !ok2 {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to publish config %s/%s", group, dataId)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Successfully published config: %s (group: %s, type: %s)", dataId, group, configType)), nil
	})

	s.AddTool(mcp.Tool{
		Name: "update_config",
		Description: "" +
			"修改 Nacos 中已存在的配置。\n" +
			"Modify an EXISTING configuration in Nacos.\n\n" +
			"IMPORTANT: This tool will FAIL if the dataId does not already exist. To create a new config, use publish_config instead.\n\n" +
			"configType must be one of: yaml, json, properties, toml, text.\n" +
			"The dataId filename extension should match the configType (e.g., mysql.yaml → yaml).\n\n" +
			"Set dryRun=true to see a side-by-side comparison (current content vs new content) without writing — always recommended before the final apply.",
		InputSchema: mcp.ToolInputSchema{
			Type: "object",
			Properties: map[string]interface{}{
				"dataId": map[string]interface{}{
					"type":        "string",
					"description": "要修改的配置标识符，必须已存在于目标 group 中。\nData ID of the config to modify. Must already exist in the target group.",
				},
				"group": map[string]interface{}{
					"type":        "string",
					"description": "配置所属分组。\nGroup of the config to modify.",
					"default":     "DEFAULT_GROUP",
				},
				"content": map[string]interface{}{
					"type":        "string",
					"description": "新的配置内容，将完整替换旧内容。\nNew content that will fully replace the existing config.",
				},
				"configType": map[string]interface{}{
					"type":        "string",
					"description": "配置类型，必须是以下之一：yaml, json, properties, toml, text。\nConfig type, must be one of: yaml, json, properties, toml, text.",
					"default":     "yaml",
				},
				"dryRun": map[string]interface{}{
					"type":        "boolean",
					"description": "设为 true 时获取当前内容并与新内容对比展示，不实际写入。建议修改前先用此模式确认变更。\nSet true to preview old-vs-new diff without writing. Always recommended before final apply.",
					"default":     false,
				},
			},
			Required: []string{"dataId", "content"},
		},
	}, func(arguments map[string]interface{}) (*mcp.CallToolResult, error) {
		dataId, _ := arguments["dataId"].(string)
		group, ok := arguments["group"].(string)
		if !ok || group == "" {
			group = "DEFAULT_GROUP"
		}
		content, _ := arguments["content"].(string)
		configType, ok := arguments["configType"].(string)
		if !ok || configType == "" {
			configType = "yaml"
		}
		dryRun, _ := arguments["dryRun"].(bool)

		if dryRun {
			oldContent, err := client.GetConfig(dataId, group)
			if err != nil {
				return mcp.NewToolResultError(fmt.Sprintf("Error fetching existing config for dry-run: %v", err)), nil
			}
			return mcp.NewToolResultText(fmt.Sprintf(
				"[dry-run] Would update config: %s (group: %s, type: %s)\n\n--- Current content ---\n%s\n\n--- New content ---\n%s",
				dataId, group, configType, oldContent, content,
			)), nil
		}

		ok2, err := client.UpdateConfig(dataId, group, content, configType)
		if err != nil {
			return mcp.NewToolResultError(fmt.Sprintf("Error: %v", err)), nil
		}

		if !ok2 {
			return mcp.NewToolResultError(fmt.Sprintf("Failed to update config %s/%s", group, dataId)), nil
		}

		return mcp.NewToolResultText(fmt.Sprintf("Successfully updated config: %s (group: %s, type: %s)", dataId, group, configType)), nil
	})
}

func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

func getEnvBool(key string, defaultValue bool) bool {
	value := os.Getenv(key)
	if value == "" {
		return defaultValue
	}
	b, err := strconv.ParseBool(value)
	if err != nil {
		log.Printf("invalid bool value for %s=%q, using default %v", key, value, defaultValue)
		return defaultValue
	}
	return b
}
