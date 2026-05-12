package main

import (
	"errors"
	"fmt"
)

// SDKVersion selects which nacos-sdk-go major version to use.
type SDKVersion string

const (
	SDKVersionV1 SDKVersion = "v1"
	SDKVersionV2 SDKVersion = "v2"
)

// NacosConfig holds the configuration for Nacos client
type NacosConfig struct {
	ServerAddr string
	Namespace  string
	Username   string
	Password   string
	ReadOnly   bool
	SDKVersion SDKVersion
}

// ConfigItem is a single entry in a config page.
type ConfigItem struct {
	DataId string
	Group  string
}

// ConfigPage is a paginated list of config items, abstracted across SDK versions.
type ConfigPage struct {
	TotalCount int
	PageItems  []ConfigItem
}

// NacosClient is the SDK-version-agnostic interface used by the MCP tools.
type NacosClient interface {
	IsReadOnly() bool
	GetConfig(dataId, group string) (string, error)
	SearchConfig(search, group string) (*ConfigPage, error)
	ListConfigs(group string, pageNo, pageSize int) (*ConfigPage, error)
	PublishConfig(dataId, group, content, configType string) (bool, error)
	UpdateConfig(dataId, group, content, configType string) (bool, error)
	Close()
}

// ErrReadOnly is returned when a write operation is attempted in read-only mode
var ErrReadOnly = errors.New("nacos-mcp is running in read-only mode: publishing and modifying configs are not allowed")

// NewNacosClient dispatches to the right SDK implementation based on config.SDKVersion.
func NewNacosClient(config *NacosConfig) (NacosClient, error) {
	switch config.SDKVersion {
	case SDKVersionV1:
		return newNacosClientV1(config)
	case "", SDKVersionV2:
		return newNacosClientV2(config)
	default:
		return nil, fmt.Errorf("unsupported NACOS_SDK_VERSION %q (expected v1 or v2)", config.SDKVersion)
	}
}
