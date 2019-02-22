package gate

import (
	"github.com/Sunmxt/linker-im/log"
	"github.com/Sunmxt/linker-im/server"
	sc "github.com/Sunmxt/linker-im/server/svc/client"
	"github.com/Sunmxt/linker-im/utils/pool"
	"net"
	"net/rpc"
)

type ServiceDripInterface struct {
	Name    string
	Addr    string
	RPCPath string
}

func OpenServiceNode(id server.NodeID, name, addr, rpcPath string, maxConcurrentRequest, maxConnection int) *server.RPCNode {
	return server.OpenRPCNode(id, name, &ServiceDripInterface{
		Name:    name,
		Addr:    addr,
		RPCPath: rpcPath,
	}, maxConcurrentRequest, maxConnection)
}

func (i *ServiceDripInterface) Keepalive(client *server.RPCClient, event chan *server.RPCNodeEvent) error {
	_, err := (*sc.ServiceClient)(client).Echo("ping")
	return err
}

// Pool interfaces
func (i *ServiceDripInterface) Healthy(x interface{}, err error) bool {
	netErr, isNetErr := err.(net.Error)
	return !(err == rpc.ErrShutdown || (isNetErr && !netErr.Timeout()))
}

func (i *ServiceDripInterface) New() (interface{}, error) {
	client, err := rpc.DialHTTPPath("tcp", i.Addr, i.RPCPath)
	if err != nil {
		log.Info0("Failed to connect service endpoint \"" + i.Addr + "\": " + err.Error())
		return nil, err
	}
	return client, nil
}

func (i *ServiceDripInterface) Destroy(x interface{}) {
	client, ok := x.(*rpc.Client)
	if !ok {
		log.Fatalf("Try to destroy object with unexcepted type. (%v)", x)
	}
	client.Close()
	log.Info0("Service RPC client closed.")
}

func (i *ServiceDripInterface) Notify(ctx *pool.NotifyContext) {
	switch ctx.Event {
	case pool.POOL_NEW:
		log.Info0("Pool created for service node \"" + i.Name + "\".")
	case pool.POOL_NEW_DRIP:
		log.Infof0("New connection for service node \""+i.Name+"\". [dripCount = %v]", ctx.DripCount)
	case pool.POOL_DESTROY_DRIP:
		log.Infof0("Close connection of service node \""+i.Name+"\". [dripCount = %v]", ctx.DripCount)
	case pool.POOL_REMOVE_DRIP:
		log.Infof0("Remove connection from pool of service node \""+i.Name+"\". [dripCount = %v]", ctx.DripCount)
	case pool.POOL_NEW_DRIP_FAILURE:
		log.Infof0("Failure occurs when connect to service node \""+i.Name+"\", [dripCoun = %v]", ctx.DripCount)
	}

	log.DebugLazy(func() string { return ctx.String() })

}
