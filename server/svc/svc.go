package svc

import (
	"fmt"
	"github.com/Sunmxt/linker-im/log"
	svcRPC "github.com/Sunmxt/linker-im/server/svc/rpc"
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
	rpcRuntime := svcRPC.ServiceRPCRuntime{
		NodeID: svcRPC.NewNodeID(),
	}
	log.Infof0("Node ID: %v", rpcRuntime.NodeID.String())

	// Register all RPC ports.
	rpcServer.Register(rpcRuntime)

	// Mux
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", Healthz)
	mux.Handle(svcRPC.RPC_PREFIX, rpcServer)

	// Serve RPC
	switch Config.Endpoint.Scheme {
	case "tcp":
		httpServer := &http.Server{
			Addr: Config.Endpoint.AuthorityString(),
			Handler: log.TagLogHandler(mux, map[string]interface{}{
				"entity": "rpc",
			}),
		}
		rpcServer.HandleHTTP(svcRPC.RPC_PATH, svcRPC.RPC_DEBUG_PATH)
		log.Infof0("RPC Serve at %v", Config.Endpoint.String())
		err = httpServer.ListenAndServe()
	default:
		err = fmt.Errorf("Not supported rpc scheme: %v", Config.Endpoint.Scheme)
	}

	return err
}

func Main() {
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
