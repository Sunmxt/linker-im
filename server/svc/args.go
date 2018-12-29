package svc

import (
	"flag"
	"fmt"
	"github.com/Sunmxt/linker-im/log"
	"github.com/Sunmxt/linker-im/utils/cmdline"
)

type ServiceOptions struct {
	// Log level
	LogLevel *cmdline.UintValue

	// Redis endpoint.
	// Used for session caching.
	RedisEndpoint *cmdline.NetEndpointValue

	// RPC endpoint. Connected by Linker Gateway.
	Endpoint *cmdline.NetEndpointValue

	// Redis prefix.
	// All the name of redis key will be add prefix to identiting application.
	// Redis prefix of all Linker Service nodes should be same.
	RedisPrefix *cmdline.StringValue

	// Cache timeout.
	CacheTimeout *cmdline.UintValue

	// Persistent storage to persist IM sessions.
	// All messages could be saved only when persistent storage provided.
	PersistStorageEndpoint *cmdline.NetEndpointValue

	// Disable message persisting.
	// DisableMessagePersist   *cmdline.BoolValue
}

func (opt *ServiceOptions) SetDefault() error {
	if opt.Endpoint.Port == 0 {
		return fmt.Errorf("Endpoint port not specified")
	}
	if opt.Endpoint.Host == "" {
		return fmt.Errorf("Endpoint host not specified.")
	}
	if opt.Endpoint.Scheme == "" {
		opt.Endpoint.Scheme = "tcp"
	}
	if opt.PersistStorageEndpoint.String() != "" {
		return fmt.Errorf("Session persisting not implemented.")
	}
	if opt.PersistStorageEndpoint.String() != "" && opt.CacheTimeout.Value != 0 {
		log.Warnf("No persistent storage endpoint. Session may be lost in %v milliseconds.", opt.CacheTimeout.Value)
	}
	return nil
}

func configureParse() (*ServiceOptions, error) {
	var err error = nil
	var redisEndpoint, RPCEndpoint, persistEndpoint *cmdline.NetEndpointValue

	const FLAGS_CREATING_FAILURE = "Flag value creating failure: %v"

	if RPCEndpoint, err = cmdline.NewNetEndpointValueDefault([]string{"http", "tcp"}, "0.0.0.0:12360"); err != nil {
		log.Panicf(FLAGS_CREATING_FAILURE, err.Error())
		return nil, err
	}
	if redisEndpoint, err = cmdline.NewNetEndpointValueDefault([]string{"tcp"}, "127.0.0.1:2379"); err != nil {
		log.Panicf(FLAGS_CREATING_FAILURE, err.Error())
		return nil, err
	}
	if persistEndpoint, err = cmdline.NewNetEndpointValueDefault([]string{"mongo"}, ""); err != nil {
		log.Panicf(FLAGS_CREATING_FAILURE, err.Error())
		return nil, err
	}

	options := &ServiceOptions{
		LogLevel:               cmdline.NewUintValueDefault(0),
		CacheTimeout:           cmdline.NewUintValueDefault(0),
		Endpoint:               RPCEndpoint,
		RedisEndpoint:          redisEndpoint,
		RedisPrefix:            cmdline.NewStringValueDefault("linker_svc"),
		PersistStorageEndpoint: persistEndpoint,
	}

	flag.Var(options.LogLevel, "log-level", "Log level.")
	flag.Var(options.Endpoint, "endpoint", "RPC bing endpoint.")
	flag.Var(options.RedisEndpoint, "redis-endpoint", "Redis endpoint used for session caching.")
	flag.Var(options.CacheTimeout, "cache-timeout", "Session cache timeout.")
	flag.Var(options.PersistStorageEndpoint, "persist-endpoint", "Storage endpoint to persist session")

	flag.Parse()

	if err = options.SetDefault(); err != nil {
		return nil, err
	}

	return options, err
}
