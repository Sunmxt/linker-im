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
	return gate.Hub.Push(msgs.Gups)
}

func Health(writer http.ResponseWriter, req *http.Request) {
	io.WriteString(writer, "ok")
}

func (g *Gate) ServeRPC() {
	if g.config.RPCEndpoint.Scheme != "tcp" {
		g.fatal <- errors.New("Not supported network type: " + g.config.RPCEndpoint.Scheme)
	}

	log.Info0("RPC Serving...")
	if err := g.RPC.ListenAndServe(); err != nil {
		log.Error("RPC Server failure: " + err.Error())
		g.fatal <- err
	}
}

func (g *Gate) InitRPC() error {
	rpc := rpc.NewServer()
	rpc.Register(GateRPC{})

	// Mux
	mux := http.NewServeMux()

	log.Info0("Register RPC health-check endpoint at \"/healthz\"")
	mux.HandleFunc("/healthz", Health)

	log.Info0("Register RPC endpoint at \"" + proto.RPC_PATH + "\"")
	mux.Handle(proto.RPC_PATH, rpc)

	g.RPC = &http.Server{
		Addr:    g.config.RPCEndpoint.AuthorityString(),
		Handler: mux,
	}
	log.Info0("RPC Serve at \"" + g.config.RPCEndpoint.String() + "\".")

	return nil
}
