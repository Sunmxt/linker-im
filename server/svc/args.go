package svc

import (
	"flag"
	"fmt"
	ilog "github.com/Sunmxt/linker-im/log"
	"github.com/Sunmxt/linker-im/utils/cmdline"
)

type ServiceOptions struct {
	// Log level
	LogLevel *cmdline.UintValue

	// Redis endpoint.
	// Used for session caching.
	RedisEndpoint *cmdline.NetEndpointValue

	// RPC endpoint. Connected by Linker Gateway and other conpoments.
	Endpoint *cmdline.NetEndpointValue

    // RPC Publish.
    RPCPublish  *cmdline.NetEndpointValue

	// Redis prefix.
	// All the name of redis key will be add prefix.
	// Redis prefix of all Linker Service nodes should be same.
	RedisPrefix *cmdline.StringValue

	// Cache timeout.
	// 0 means infinite timeout.
	CacheTimeout *cmdline.UintValue


	// Persistent storage endpoint to persist sessions and messages.
	//PersistStorageEndpoint *cmdline.NetEndpointValue

	// Do not persist sessions.
	// Ignored when no persistent endpoint specified.
	//DisableSessionPersist *cmdline.BoolValue

	// Do not persist messages.
	// Ignored when no persistent endpoint specified.
	//DisableMessagePersist *cmdline.BoolValue

	// Reject all session operations when persistent storage fails.
	// Ignored when no persistent endpoint specified.
	///FailOnPersistFailure *cmdline.BoolValue

	// Asynchronous session persisting.
	//AsyncSessionPersist *cmdline.BoolValue

	// Asynchronous message persisting.
	//AsyncMessagePersist *cmdline.BoolValue
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
	//if opt.PersistStorageEndpoint.String() != "" {
	//	return fmt.Errorf("Session persisting not implemented.")
	//}
	//if opt.PersistStorageEndpoint.String() != "" && opt.CacheTimeout.Value != 0 {
	//	ilog.Warnf("No persistent storage endpoint. Session may be lost in %v milliseconds.", opt.CacheTimeout.Value)
	//}
	if opt.CacheTimeout.Value < 0 {
		ilog.Warnf("Cache timeout should not be nagtive. Set to 0.", opt.CacheTimeout.Value)
	}
    if opt.RPCPublish.Port == 0 || opt.RPCPublish.Port > 0xFFFF {
        opt.RPCPublish.Port = opt.Endpoint.Port
    }
    if opt.RPCPublish.Host == "localhost" || opt.RPCPublish.Host == "127.0.0.1" {
        ilog.Warn("Publish local address: " + opt.RPCPublish.String())
    }
	return nil
}

func configureParse() (*ServiceOptions, error) {
	var err error = nil
	var redisEndpoint, RPCEndpoint, publish *cmdline.NetEndpointValue

	const FLAGS_CREATING_FAILURE = "Flag value creating failure: %v"

	if RPCEndpoint, err = cmdline.NewNetEndpointValueDefault([]string{"http", "tcp"}, "0.0.0.0:12361"); err != nil {
		ilog.Panicf(FLAGS_CREATING_FAILURE, err.Error())
		return nil, err
	}
	if redisEndpoint, err = cmdline.NewNetEndpointValueDefault([]string{"tcp"}, "127.0.0.1:6379"); err != nil {
		ilog.Panicf(FLAGS_CREATING_FAILURE, err.Error())
		return nil, err
	}
	//if persistEndpoint, err = cmdline.NewNetEndpointValueDefault([]string{"mongo"}, ""); err != nil {
	//	ilog.Panicf(FLAGS_CREATING_FAILURE, err.Error())
	//	return nil, err
	//}
    if publish, err = cmdline.NewNetEndpointValueDefault([]string{"tcp"}, "127.0.0.1:0"); err != nil {
		ilog.Panicf(FLAGS_CREATING_FAILURE, err.Error())
		return nil, err
    }

	options := &ServiceOptions{
		LogLevel:               cmdline.NewUintValueDefault(0),
		CacheTimeout:           cmdline.NewUintValueDefault(0),
		Endpoint:               RPCEndpoint,
		RedisEndpoint:          redisEndpoint,
		RedisPrefix:            cmdline.NewStringValueDefault("linker_svc"),
		//PersistStorageEndpoint: persistEndpoint,
		//DisableSessionPersist:  cmdline.NewBoolValueDefault(false),
		//DisableMessagePersist:  cmdline.NewBoolValueDefault(false),
		//FailOnPersistFailure:   cmdline.NewBoolValueDefault(true),
		//AsyncSessionPersist:    cmdline.NewBoolValueDefault(false),
		//AsyncMessagePersist:    cmdline.NewBoolValueDefault(true),
        RPCPublish:                publish,
	}

	flag.Var(options.LogLevel, "log-level", "Log level.")
	flag.Var(options.Endpoint, "endpoint", "RPC bing endpoint.")
	flag.Var(options.RedisEndpoint, "redis-endpoint", "Redis endpoint used for session caching.")
	flag.Var(options.CacheTimeout, "cache-timeout", "Session cache timeout.")
	//flag.Var(options.PersistStorageEndpoint, "persist-endpoint", "Storage endpoint to persist session")
	//flag.Var(options.DisableMessagePersist, "disable-message-persist", "Do not persist messages.")
	//flag.Var(options.DisableSessionPersist, "disable-session-persist", "Do not persist sessions.")
	//flag.Var(options.FailOnPersistFailure, "fail-on-persist-failure", "Reject all session options when persistent storage fails.")
	//flag.Var(options.AsyncMessagePersist, "async-message-persist", "Persist messages asynchronously.")
	//flag.Var(options.AsyncSessionPersist, "async-session-persist", "Persist session asynchronously.")
    flag.Var(options.RPCPublish, "rpc-publish", "Published RPC endpoint.")

	flag.Parse()

	if err = options.SetDefault(); err != nil {
		return nil, err
	}

    ilog.Info0("Configurations:")
	flag.VisitAll(func(fl *flag.Flag) {
		ilog.Info0("-" + fl.Name + "=" + fl.Value.String())
	})

	return options, err
}
