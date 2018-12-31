package gate

import (
	"flag"
	"fmt"
	"github.com/Sunmxt/linker-im/log"
	config "github.com/Sunmxt/linker-im/server/gate/config"
	"github.com/Sunmxt/linker-im/utils/cmdline"
	yaml "gopkg.in/yaml.v2"
	"io/ioutil"
)

type GatewayOptions struct {
	ExternalConfig   *cmdline.StringValue
	LogLevel         *cmdline.UintValue
	PublicManagement *cmdline.BoolValue
	ManageEndpoint   *cmdline.NetEndpointValue
	APIEndpoint      *cmdline.NetEndpointValue
	RedisEndpoint    *cmdline.NetEndpointValue
	ServiceEndpoints *cmdline.NetEndpointSetValue
}

func (options *GatewayOptions) SetDefaultFromConfigure(cfg *config.GatewayConfigure) error {
	if options.LogLevel.IsDefault {
		options.LogLevel.Value = cfg.LogLevel
	}

	if options.PublicManagement.IsDefault {
		options.PublicManagement.Value = cfg.IMEnableManagementAPI
	}

	if options.ManageEndpoint.IsDefault && cfg.ManageEndpoint != "" {
		if err := options.ManageEndpoint.Set(cfg.ManageEndpoint); err != nil {
			return err
		}
	}

	if options.APIEndpoint.IsDefault && cfg.IMAPIEndpoint != "" {
		if err := options.APIEndpoint.Set(cfg.IMAPIEndpoint); err != nil {
			return err
		}
	}

	if options.ServiceEndpoints.IsDefault && cfg.ServiceEndpoints != "" {
		if err := options.ServiceEndpoints.Set(cfg.ServiceEndpoints); err != nil {
			return err
		}
	}

	return nil
}

func (options *GatewayOptions) SetDefault() error {
	if options.ServiceEndpoints.String() == "" {
		return fmt.Errorf("No service node found. (See \"-service-endpoints\")")
	}
	if options.RedisEndpoint.Host == "" {
		return fmt.Errorf("Redis endpoint hosts hould not be empty. (See \"-redis-endpoint\")")
	}
	if !options.RedisEndpoint.HasPort {
		return fmt.Errorf("Redis endpoint port should be specified. (See \"-redis-endpoint\")")
	}
	if options.RedisEndpoint.Port == 0 || options.RedisEndpoint.Port > 0xFFFF {
		return fmt.Errorf("Redis endpoint port should not be %v. (See \"-redis-endpoint\")", options.RedisEndpoint.Port)
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
		ExternalConfig:   cmdline.NewStringValue(),
		LogLevel:         cmdline.NewUintValueDefault(0),
		PublicManagement: cmdline.NewBoolValueDefault(false),
		ManageEndpoint:   manage_endpoint,
		APIEndpoint:      api_endpoint,
		RedisEndpoint:    redis_endpoint,
		ServiceEndpoints: serviceEndpoints,
	}

	flag.Var(options.ExternalConfig, "config", "Configure YAML.")
	flag.Var(options.LogLevel, "log-level", "Log level.")
	flag.Var(options.PublicManagement, "enable-public-management", "Enable management API on public endpoint.")
	flag.Var(options.APIEndpoint, "endpoint", "Public API binding Endpoint.")
	flag.Var(options.ManageEndpoint, "manage-endpoint", "Manage API Endpoint.")
	flag.Var(options.RedisEndpoint, "redis-endpoint", "Redis cache endpoint.")
	flag.Var(options.ServiceEndpoints, "service-endpoints", "Service node endpoints.")

	flag.Parse()

	// Load configure when external yaml is given.
	if options.ExternalConfig.Value != "" {
		var config_content []byte
		external_config := &config.GatewayConfigure{
			LogLevel:              0,
			IMAPIEndpoint:         "",
			IMEnableManagementAPI: false,
			ManageEndpoint:        "",
			RedisEndpoint:         "",
			ServiceEndpoints:      "",
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
	return options, err
}