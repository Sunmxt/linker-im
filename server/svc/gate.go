package svc

import (
	"github.com/Sunmxt/linker-im/log"
	"github.com/Sunmxt/linker-im/server"
	sc "github.com/Sunmxt/linker-im/server/gate/client"
	"github.com/Sunmxt/linker-im/utils/pool"
	"net"
	"net/rpc"
)

type GateDripInterface struct {
	Name    string
	Addr    string
	RPCPath string
}

func OpenGateNode(id server.NodeID, name, addr, rpcPath string, maxConcurrentRequest, maxConnection int) *server.RPCNode {
	return server.OpenRPCNode(id, name, &GateDripInterface{
		Name:    name,
		Addr:    addr,
		RPCPath: rpcPath,
	}, maxConcurrentRequest, maxConnection)
}

func (i *GateDripInterface) Keepalive(client *server.RPCClient, event chan *server.RPCNodeEvent) error {
	_, err := (*sc.GateClient)(client).Echo("ping")
	return err
}

// Pool interfaces
func (i *GateDripInterface) Healthy(x interface{}, err error) bool {
	netErr, isNetErr := err.(net.Error)
	return !(err == rpc.ErrShutdown || (isNetErr && !netErr.Timeout()))
}

func (i *GateDripInterface) New() (interface{}, error) {
	client, err := rpc.DialHTTPPath("tcp", i.Addr, i.RPCPath)
	if err != nil {
		log.Info0("Failed to connect gate endpoint \"" + i.Addr + "\": " + err.Error())
		return nil, err
	}
	return client, nil
}

func (i *GateDripInterface) Destroy(x interface{}) {
	client, ok := x.(*rpc.Client)
	if !ok {
		log.Fatalf("Try to destroy object with unexcepted type. (%v)", x)
	}
	client.Close()
	log.Info0("Service RPC client closed.")
}

func (i *GateDripInterface) Notify(ctx *pool.NotifyContext) {
	switch ctx.Event {
	case pool.POOL_NEW:
		log.Info0("Pool created for gate node \"" + i.Name + "\".")
	case pool.POOL_NEW_DRIP:
		log.Infof0("New connection for gate node \""+i.Name+"\". [dripCount = %v]", ctx.DripCount)
	case pool.POOL_DESTROY_DRIP:
		log.Infof0("Close connection of gate node \""+i.Name+"\". [dripCount = %v]", ctx.DripCount)
	case pool.POOL_REMOVE_DRIP:
		log.Infof0("Remove connection from pool of gate node \""+i.Name+"\". [dripCount = %v]", ctx.DripCount)
	case pool.POOL_NEW_DRIP_FAILURE:
		log.Infof0("Failure occurs when connect to gate node \""+i.Name+"\", [dripCoun = %v]", ctx.DripCount)
	}

	log.DebugLazy(func() string { return ctx.String() })

}
