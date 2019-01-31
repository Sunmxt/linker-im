package gate

import (
	"errors"
	"github.com/Sunmxt/linker-im/log"
	"github.com/Sunmxt/linker-im/proto"
	"io"
	"net/http"
	"net/rpc"
)

type GateRPC struct{}

func (r GateRPC) Push(msgs *proto.MessagePushArguments, reply *struct{}) error {
	hub, err := GetHub()
	if err != nil {
		return err
	}
	return hub.Push(msgs.Gups)
}

func Health(writer http.ResponseWriter, req *http.Request) {
	io.WriteString(writer, "ok")
}

func NewServiceMux() (*http.ServeMux, error) {
	rpcServer := rpc.NewServer()
	rpcServer.Register(GateRPC{})

	// Mux
	mux := http.NewServeMux()

	log.Info0("Register health-check HTTP endpoint at \"/healthz\"")
	mux.HandleFunc("/healthz", Health)

	log.Info0("Register RPC HTTP endpoint at \"" + proto.RPC_PATH + "\"")
	mux.Handle(proto.RPC_PATH, rpcServer)

	return mux, nil
}

func ServeRPC() {
	mux, err := NewServiceMux()
	defer func() {
		if err != nil {
			log.Error("RPC Server failure: " + err.Error())
		}
	}()
	if err != nil {
		return
	}

	switch Config.RPCEndpoint.Scheme {
	case "tcp":
		server := http.Server{
			Addr:    Config.RPCEndpoint.AuthorityString(),
			Handler: mux,
		}
		log.Info0("RPC Serve at \"" + Config.RPCEndpoint.String() + "\"")
		err = server.ListenAndServe()
	default:
		err = errors.New("Not supported network type: " + Config.RPCEndpoint.Scheme)
	}
}
