package svc

import (
	"fmt"
	"github.com/Sunmxt/linker-im/log"
	"github.com/Sunmxt/linker-im/proto"
	"github.com/Sunmxt/linker-im/server"
	"io"
	"net/http"
	"net/rpc"
)

var Config *ServiceOptions

func LogConfigure() {
	log.Infof0("-endpoint=%v", Config.Endpoint.String())
	log.Infof0("-log-level=%v", Config.LogLevel.String())
	log.Infof0("-redis-endpoint=%v", Config.LogLevel.String())
	log.Infof0("-redis-prefix=%v", Config.RedisPrefix.String())
	log.Infof0("-persist-endpoint=%v", Config.PersistStorageEndpoint.String())
	log.Infof0("-cache-timeout=%v", Config.CacheTimeout.Value)
}

func Healthz(w http.ResponseWriter, req *http.Request) {
	io.WriteString(w, "ok")
}

func ServeRPC() error {
	var err error

	rpcServer := rpc.NewServer()
	rpcRuntime := ServiceRPC{
		NodeID: server.NewNodeID(),
	}
	log.Infof0("Node ID: %v", rpcRuntime.NodeID.String())

	// Register all RPC ports.
	rpcServer.Register(rpcRuntime)

	// Mux
	healthCheckMux := http.NewServeMux()
	healthCheckMux.HandleFunc("/", Healthz)
	mux := http.NewServeMux()
	mux.Handle("/healthz", log.TagLogHandler(healthCheckMux, map[string]interface{}{
		"entity": "health-check",
	}))
	mux.Handle(proto.RPC_PATH, rpcServer)

	// Serve RPC
	switch Config.Endpoint.Scheme {
	case "tcp":
		httpServer := &http.Server{
			Addr:    Config.Endpoint.AuthorityString(),
			Handler: mux,
		}
		log.Infof0("RPC Serve at %v", Config.Endpoint.String())
		err = httpServer.ListenAndServe()
	default:
		err = fmt.Errorf("Not supported rpc scheme: %v", Config.Endpoint.Scheme)
	}

	return err
}

func Main() {
	fmt.Println("Service node of Linker IM.")
	opt, err := configureParse()
	if opt == nil {
		log.Fatalf("%v", err.Error())
		return
	}

	Config = opt
	log.Info0("Linker IM Service start.")
	LogConfigure()

	// Log level
	log.Infof0("Log level: %v", Config.LogLevel.Value)
	log.SetGlobalLogLevel(Config.LogLevel.Value)

	// Serve RPC
	if err = ServeRPC(); err != nil {
		log.Fatalf("RPC Failure: %v", err.Error())
	}
}
