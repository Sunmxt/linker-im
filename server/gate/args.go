package gate

import (
	"flag"
	"fmt"
	config "github.com/Sunmxt/linker-im/config"
	"github.com/Sunmxt/linker-im/log"
	"github.com/Sunmxt/linker-im/utils/cmdline"
	yaml "gopkg.in/yaml.v2"
	"io/ioutil"
)

type GatewayOptions struct {
	ExternalConfig *cmdline.StringValue
	LogLevel       *cmdline.UintValue

	ManageEndpoint *cmdline.NetEndpointValue

	// Endpoint to bind and serve HTTP API.
	APIEndpoint *cmdline.NetEndpointValue

	// Redis endpoint.
	RedisEndpoint *cmdline.NetEndpointValue

	// Redis prefix.
	RedisPrefix *cmdline.StringValue

	// List of service endpoints.
	ServiceEndpoints *cmdline.NetEndpointSetValue

	// Period to check state of service endpoint.
	// Unhealthy endpoints will be disable automatically.
	KeepalivePeriod *cmdline.UintValue

	// Timeslice length to aggregate messages.
	// All messages received within the period will be grouped and sent within one response.
	// Timeslice will not be longer then timeout specified by client.
	MessageBulkTime *cmdline.UintValue

    // Active time.
    ActiveTimeout *cmdline.UintValue

	// Debug mode
	// More information will be reported to clients when debug mode is on.
	DebugMode *cmdline.BoolValue
}

func (options *GatewayOptions) SetDefaultFromConfigure(cfg *config.GatewayConfigure) error {
	if options.LogLevel.IsDefault {
		options.LogLevel.Value = cfg.LogLevel
	}
	if options.ManageEndpoint.IsDefault && cfg.Manage.Endpoint != "" {
		if err := options.ManageEndpoint.Set(cfg.Manage.Endpoint); err != nil {
			return err
		}
	}

	if options.APIEndpoint.IsDefault && cfg.HTTPConfig.Endpoint != "" {
		if err := options.APIEndpoint.Set(cfg.HTTPConfig.Endpoint); err != nil {
			return err
		}
	}
	if options.ServiceEndpoints.IsDefault && cfg.SVCConfig.Endpoints != nil && len(cfg.SVCConfig.Endpoints) > 0 {
		options.ServiceEndpoints.IsDefault = false
		for k, v := range cfg.SVCConfig.Endpoints {
			ep, err := cmdline.NewNetEndpointValueDefault(options.ServiceEndpoints.ValidSchemes, v)
			if err != nil {
				return err
			}
			options.ServiceEndpoints.Endpoints[k] = ep
		}
	}
	if options.KeepalivePeriod.IsDefault {
		options.KeepalivePeriod.Value = cfg.SVCConfig.KeepalivePeriod
	}
	if options.DebugMode.IsDefault {
		options.DebugMode.Value = cfg.Debug
	}
	if options.RedisPrefix.IsDefault {
		options.RedisPrefix.Value = cfg.RedisPrefix
	}
    if options.ActiveTimeout.IsDefault {
        options.ActiveTimeout.Value = cfg.HTTPConfig.ActiveTime
    }
	return nil
}

func (options *GatewayOptions) SetDefault() error {
	if options.ServiceEndpoints.String() == "" {
		return fmt.Errorf("No service node found. (See \"-service-endpoints\")")
	}
	if options.RedisEndpoint.Host == "" {
		return fmt.Errorf("Redis endpoint hosts should not be empty. (See \"-redis-endpoint\")")
	}
	if !options.RedisEndpoint.HasPort {
		return fmt.Errorf("Redis endpoint port should be specified. (See \"-redis-endpoint\")")
	}
	if options.RedisEndpoint.Port == 0 || options.RedisEndpoint.Port > 0xFFFF {
		return fmt.Errorf("Redis endpoint port should not be %v. (See \"-redis-endpoint\")", options.RedisEndpoint.Port)
	}
	if options.KeepalivePeriod.Value == 0 {
		return fmt.Errorf("Keepalive period should not be %v. (See \"-keepalive-period\")", options.RedisEndpoint.Port)
	}
	if options.RedisEndpoint.Scheme == "" {
		options.RedisEndpoint.Scheme = "tcp"
	}
	return nil
}

func configureParse() (*GatewayOptions, error) {
	var err error = nil
	var api_endpoint, manage_endpoint, redis_endpoint *cmdline.NetEndpointValue
	var serviceEndpoints *cmdline.NetEndpointSetValue

	if manage_endpoint, err = cmdline.NewNetEndpointValueDefault([]string{"tcp", "http", "https"}, "127.0.0.1:12361"); err != nil {
		log.Panicf("Flag value creating failure: %v", err.Error())
		return nil, err
	}
	if api_endpoint, err = cmdline.NewNetEndpointValueDefault([]string{"tcp", "http", "https"}, "0.0.0.0:12360"); err != nil {
		log.Panicf("Flag value creating failure: %v", err.Error())
		return nil, err
	}
	if redis_endpoint, err = cmdline.NewNetEndpointValueDefault([]string{"tcp"}, ""); err != nil {
		log.Panicf("Flag value creating failure: %v", err.Error())
		return nil, err
	}
	if serviceEndpoints, err = cmdline.NewNetEndpointSetValueDefault([]string{"tcp"}, ""); err != nil {
		log.Panicf("Flag value creating failure: %v", err.Error())
		return nil, err
	}

	options := &GatewayOptions{
		ExternalConfig:            cmdline.NewStringValue(),
		LogLevel:                  cmdline.NewUintValueDefault(0),
		KeepalivePeriod:           cmdline.NewUintValueDefault(10),
		ManageEndpoint:            manage_endpoint,
		APIEndpoint:               api_endpoint,
		RedisEndpoint:             redis_endpoint,
		RedisPrefix:               cmdline.NewStringValueDefault("linker"),
		ServiceEndpoints:          serviceEndpoints,
		MessageBulkTime: cmdline.NewUintValueDefault(50),
        ActiveTimeout:             cmdline.NewUintValueDefault(5000),
		DebugMode:                 cmdline.NewBoolValueDefault(false),
	}

	flag.Var(options.ExternalConfig, "config", "Configure YAML.")
	flag.Var(options.LogLevel, "log-level", "Log level.")
	flag.Var(options.APIEndpoint, "endpoint", "Public API binding Endpoint.")
	flag.Var(options.ManageEndpoint, "manage-endpoint", "Manage API Endpoint.")
	flag.Var(options.RedisEndpoint, "redis-endpoint", "Redis cache endpoint.")
	flag.Var(options.RedisPrefix, "redis-prefix", "Redis cache key prefix.")
	flag.Var(options.ServiceEndpoints, "service-endpoints", "Service node endpoints.")
	flag.Var(options.KeepalivePeriod, "keepalive-period", "Keepalive period. Can not be 0.")
    flag.Var(options.ActiveTimeout, "active-timeout", "")
	flag.Var(options.DebugMode, "debug", "Enable debug mode.")

	flag.Parse()

	// Load configure when external yaml is given.
	if options.ExternalConfig.Value != "" {
		var config_content []byte
		external_config := &config.GatewayConfigure{
			LogLevel:                  0,
			MessageBulkTime: 50,
			RedisEndpoint:             "",
			RedisPrefix:               "linker",
			Debug:                     false,
			SVCConfig: config.ServiceConnectionConfigure{
				Endpoints:       make(map[string]string),
				KeepalivePeriod: 10,
			},
			HTTPConfig: config.HTTPAPIConfigure{
				Endpoint: "0.0.0.0:12360",
                ActiveTime: 5000,
			},
		}

		log.Info0("External configure: %v", options.ExternalConfig.Value)

		if config_content, err = ioutil.ReadFile(options.ExternalConfig.Value); err != nil {
			return nil, fmt.Errorf("Failed to load configure file: %v", err.Error())
		}

		if err = yaml.Unmarshal(config_content, external_config); err != nil {
			return nil, fmt.Errorf("Invalid configure format: %v", err.Error())
		}

		if err = options.SetDefaultFromConfigure(external_config); err != nil {
			return nil, fmt.Errorf("Invalid configure: %v", err.Error())
		}
	}

	if err = options.SetDefault(); err != nil {
		return nil, err
	}

    log.Info0("Configurations:")
    flag.VisitAll(func (fl *flag.Flag) {
        log.Info0("-" + fl.Name + "=" + fl.Value.String())
    })

	return options, err
}
