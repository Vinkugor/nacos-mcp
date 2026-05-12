package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	v1clients "github.com/nacos-group/nacos-sdk-go/clients"
	v1configclient "github.com/nacos-group/nacos-sdk-go/clients/config_client"
	v1constant "github.com/nacos-group/nacos-sdk-go/common/constant"
	v1model "github.com/nacos-group/nacos-sdk-go/model"
	v1vo "github.com/nacos-group/nacos-sdk-go/vo"
)

// nacosClientV1 wraps nacos-sdk-go v1 (HTTP, compatible with Nacos Server 1.x).
type nacosClientV1 struct {
	client   v1configclient.IConfigClient
	readOnly bool
}

func newNacosClientV1(config *NacosConfig) (*nacosClientV1, error) {
	addr := strings.TrimPrefix(config.ServerAddr, "https://")
	addr = strings.TrimPrefix(addr, "http://")
	host, portStr, err := net.SplitHostPort(addr)
	if err != nil {
		return nil, fmt.Errorf("invalid server address %q: %w", config.ServerAddr, err)
	}
	port, err := strconv.ParseUint(portStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid port %q: %w", portStr, err)
	}

	sc := []v1constant.ServerConfig{
		*v1constant.NewServerConfig(host, port, v1constant.WithContextPath("/nacos")),
	}

	runtimeDir := filepath.Join(os.TempDir(), "nacos-mcp")
	cc := *v1constant.NewClientConfig(
		v1constant.WithNamespaceId(config.Namespace),
		v1constant.WithUsername(config.Username),
		v1constant.WithPassword(config.Password),
		v1constant.WithTimeoutMs(10000),
		v1constant.WithNotLoadCacheAtStart(true),
		v1constant.WithLogDir(filepath.Join(runtimeDir, "log")),
		v1constant.WithCacheDir(filepath.Join(runtimeDir, "cache")),
		v1constant.WithLogLevel("warn"),
	)

	client, err := v1clients.NewConfigClient(v1vo.NacosClientParam{
		ClientConfig:  &cc,
		ServerConfigs: sc,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create nacos v1 config client: %w", err)
	}

	return &nacosClientV1{client: client, readOnly: config.ReadOnly}, nil
}

func (nc *nacosClientV1) IsReadOnly() bool { return nc.readOnly }

func (nc *nacosClientV1) GetConfig(dataId, group string) (string, error) {
	content, err := nc.client.GetConfig(v1vo.ConfigParam{
		DataId: dataId,
		Group:  group,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get config %s/%s: %w", group, dataId, err)
	}
	return content, nil
}

func (nc *nacosClientV1) SearchConfig(search, group string) (*ConfigPage, error) {
	page, err := nc.client.SearchConfig(v1vo.SearchConfigParam{
		Search:   "blur",
		DataId:   search + "*",
		Group:    group,
		PageNo:   1,
		PageSize: 100,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search config: %w", err)
	}
	return convertV1Page(page), nil
}

func (nc *nacosClientV1) ListConfigs(group string, pageNo, pageSize int) (*ConfigPage, error) {
	page, err := nc.client.SearchConfig(v1vo.SearchConfigParam{
		Search:   "accurate",
		Group:    group,
		PageNo:   pageNo,
		PageSize: pageSize,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list configs: %w", err)
	}
	return convertV1Page(page), nil
}

func (nc *nacosClientV1) PublishConfig(dataId, group, content, configType string) (bool, error) {
	if nc.readOnly {
		return false, ErrReadOnly
	}
	ok, err := nc.client.PublishConfig(v1vo.ConfigParam{
		DataId:  dataId,
		Group:   group,
		Content: content,
		Type:    v1vo.ConfigType(configType),
	})
	if err != nil {
		return false, fmt.Errorf("failed to publish config %s/%s: %w", group, dataId, err)
	}
	return ok, nil
}

func (nc *nacosClientV1) UpdateConfig(dataId, group, content, configType string) (bool, error) {
	if nc.readOnly {
		return false, ErrReadOnly
	}
	if _, err := nc.GetConfig(dataId, group); err != nil {
		return false, fmt.Errorf("failed to verify config %s/%s exists before update: %w", group, dataId, err)
	}
	return nc.PublishConfig(dataId, group, content, configType)
}

func (nc *nacosClientV1) Close() {
	// v1 IConfigClient exposes no explicit close; lifetime ends with the process.
}

func convertV1Page(page *v1model.ConfigPage) *ConfigPage {
	if page == nil {
		return &ConfigPage{}
	}
	items := make([]ConfigItem, 0, len(page.PageItems))
	for _, it := range page.PageItems {
		items = append(items, ConfigItem{DataId: it.DataId, Group: it.Group})
	}
	return &ConfigPage{TotalCount: page.TotalCount, PageItems: items}
}
