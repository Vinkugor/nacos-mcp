package main

import (
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/nacos-group/nacos-sdk-go/v2/clients"
	"github.com/nacos-group/nacos-sdk-go/v2/clients/config_client"
	"github.com/nacos-group/nacos-sdk-go/v2/common/constant"
	"github.com/nacos-group/nacos-sdk-go/v2/model"
	"github.com/nacos-group/nacos-sdk-go/v2/vo"
)

// nacosClientV2 wraps nacos-sdk-go/v2 (gRPC, requires Nacos Server >= 2.x).
type nacosClientV2 struct {
	client   config_client.IConfigClient
	readOnly bool
}

func newNacosClientV2(config *NacosConfig) (*nacosClientV2, error) {
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

	sc := []constant.ServerConfig{
		*constant.NewServerConfig(host, port, constant.WithContextPath(config.ContextPath)),
	}

	runtimeDir := filepath.Join(os.TempDir(), "nacos-mcp")
	cc := *constant.NewClientConfig(
		constant.WithNamespaceId(config.Namespace),
		constant.WithUsername(config.Username),
		constant.WithPassword(config.Password),
		constant.WithTimeoutMs(10000),
		constant.WithNotLoadCacheAtStart(true),
		constant.WithDisableUseSnapShot(true),
		constant.WithLogDir(filepath.Join(runtimeDir, "log")),
		constant.WithCacheDir(filepath.Join(runtimeDir, "cache")),
		constant.WithLogLevel("warn"),
	)

	client, err := clients.NewConfigClient(vo.NacosClientParam{
		ClientConfig:  &cc,
		ServerConfigs: sc,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create nacos v2 config client: %w", err)
	}

	return &nacosClientV2{client: client, readOnly: config.ReadOnly}, nil
}

func (nc *nacosClientV2) IsReadOnly() bool { return nc.readOnly }

func (nc *nacosClientV2) GetConfig(dataId, group string) (string, error) {
	content, err := nc.client.GetConfig(vo.ConfigParam{
		DataId: dataId,
		Group:  group,
	})
	if err != nil {
		return "", fmt.Errorf("failed to get config %s/%s: %w", group, dataId, err)
	}
	return content, nil
}

func (nc *nacosClientV2) SearchConfig(search, group string) (*ConfigPage, error) {
	page, err := nc.client.SearchConfig(vo.SearchConfigParam{
		Search:   "blur",
		DataId:   search + "*",
		Group:    group,
		PageNo:   1,
		PageSize: 100,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to search config: %w", err)
	}
	return convertV2Page(page), nil
}

func (nc *nacosClientV2) ListConfigs(group string, pageNo, pageSize int) (*ConfigPage, error) {
	page, err := nc.client.SearchConfig(vo.SearchConfigParam{
		Search:   "accurate",
		Group:    group,
		PageNo:   pageNo,
		PageSize: pageSize,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to list configs: %w", err)
	}
	return convertV2Page(page), nil
}

func (nc *nacosClientV2) PublishConfig(dataId, group, content, configType string) (bool, error) {
	if nc.readOnly {
		return false, ErrReadOnly
	}
	ok, err := nc.client.PublishConfig(vo.ConfigParam{
		DataId:  dataId,
		Group:   group,
		Content: content,
		Type:    configType,
	})
	if err != nil {
		return false, fmt.Errorf("failed to publish config %s/%s: %w", group, dataId, err)
	}
	return ok, nil
}

func (nc *nacosClientV2) UpdateConfig(dataId, group, content, configType string) (bool, error) {
	if nc.readOnly {
		return false, ErrReadOnly
	}
	if _, err := nc.GetConfig(dataId, group); err != nil {
		return false, fmt.Errorf("failed to verify config %s/%s exists before update: %w", group, dataId, err)
	}
	return nc.PublishConfig(dataId, group, content, configType)
}

func (nc *nacosClientV2) Close() {
	nc.client.CloseClient()
}

func convertV2Page(page *model.ConfigPage) *ConfigPage {
	if page == nil {
		return &ConfigPage{}
	}
	items := make([]ConfigItem, 0, len(page.PageItems))
	for _, it := range page.PageItems {
		items = append(items, ConfigItem{DataId: it.DataId, Group: it.Group})
	}
	return &ConfigPage{TotalCount: page.TotalCount, PageItems: items}
}
