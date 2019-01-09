package config

import (
//yaml "gopkg.in/yaml.v2"
)

type HTTPAPIConfigure struct {
    Endpoint        string `endpoint,omitempty`
}

type HTTPManagementAPIConfigure struct {
    Endpoint        string `endpoint,omitempty`
}

type ServiceConnectionConfigure struct {
    Endpoints       map[string]string   `endpoints,omitempty`
    KeepalivePeriod uint                `keepalive-period,omitempty`
}

type GatewayConfigure struct {
	LogLevel uint `log-level,omitempty,flow`

    MessageAggregateTimeslice uint `aggregate-timeslice,omitempty`

	RedisEndpoint           string `redis-endpoint,omitempty`

    SVCConfig             ServiceConnectionConfigure    `service,omitempty`
    HTTPConfig            HTTPAPIConfigure              `http,omitempty`
    Manage                HTTPManagementAPIConfigure    `manage,omitempty`
}
