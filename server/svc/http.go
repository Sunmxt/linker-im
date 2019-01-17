package svc

import (
	ilog "github.com/Sunmxt/linker-im/log"
	"github.com/Sunmxt/linker-im/proto"
	"github.com/Sunmxt/linker-im/server"
	"io"
	"net/http"
	"net/rpc"
)

func Healthz(w http.ResponseWriter, req *http.Request) {
	io.WriteString(w, "ok")
}

func NewServiceServeMux() (*http.ServeMux, error) {
	rpcServer := rpc.NewServer()
	rpcRuntime := ServiceRPC{
		NodeID: server.NewNodeID(),
		log:    ilog.NewLogger(),
	}
	rpcRuntime.log.Fields["entity"] = "rpc"

	ilog.Infof0("Node ID: %v", rpcRuntime.NodeID.String())

	// Register all RPC ports.
	rpcServer.Register(rpcRuntime)

	// Mux
	healthCheckMux := http.NewServeMux()
	healthCheckMux.HandleFunc("/", Healthz)
	mux := http.NewServeMux()

	// Health check
	mux.Handle("/healthz", ilog.TagLogHandler(healthCheckMux, map[string]interface{}{
		"entity": "health-check",
	}))

	// RPC
	mux.Handle(proto.RPC_PATH, rpcServer)

	return mux, nil
}
