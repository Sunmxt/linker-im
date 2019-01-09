package svc

import (
	"fmt"
	ilog "github.com/Sunmxt/linker-im/log"
)

var Config *ServiceOptions

func LogConfigure() {
	ilog.Infof0("-endpoint=%v", Config.Endpoint.String())
	ilog.Infof0("-log-level=%v", Config.LogLevel.String())
	ilog.Infof0("-redis-endpoint=%v", Config.LogLevel.String())
	ilog.Infof0("-redis-prefix=%v", Config.RedisPrefix.String())
	ilog.Infof0("-persist-endpoint=%v", Config.PersistStorageEndpoint.String())
	ilog.Infof0("-cache-timeout=%v", Config.CacheTimeout.Value)
	ilog.Infof0("-persist-endpoint=%v", Config.PersistStorageEndpoint.String())
	ilog.Infof0("-disable-message-persist=%v", Config.DisableMessagePersist.String())
	ilog.Infof0("-disable-session-persist=%v", Config.DisableSessionPersist.String())
	ilog.Infof0("-fail-on-persist-failure=%v", Config.FailOnPersistFailure.String())
	ilog.Infof0("-async-message-persist=%v", Config.AsyncMessagePersist.String())
	ilog.Infof0("-async-session-persist=%v", Config.AsyncSessionPersist.String())
}

func Main() {
	fmt.Println("Service node of Linker IM.")
	opt, err := configureParse()
	if opt == nil {
		ilog.Fatalf("%v", err.Error())
		return
	}

	Config = opt
	ilog.Info0("Linker IM Service start.")
	LogConfigure()

	// Log level
	ilog.Infof0("Log level: %v", Config.LogLevel.Value)
	ilog.SetGlobalLogLevel(Config.LogLevel.Value)

	// Register resource
	if err = RegisterResources(); err != nil {
		ilog.Fatalf("Failed to register resource: %v", err.Error())
	}

	// Serve RPC
	if err = ServeRPC(); err != nil {
		ilog.Fatalf("RPC Failure: %v", err.Error())
	}
}
