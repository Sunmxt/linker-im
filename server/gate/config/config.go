package config

import (
//yaml "gopkg.in/yaml.v2"
)

type GatewayConfigure struct {
	LogLevel uint `log-level,omitempty,flow`

	IMAPIEndpoint         string `api-endpoint,omitempty`
	IMEnableManagementAPI bool   `enable-manage-api,omitempty`
	ManageEndpoint        string `manage-endpoint,omitempty`
	RedisEndpoint         string `redis-endpoint,omitempty`
	ServiceEndpoints      string `service-endpoints,omitempty`
}
