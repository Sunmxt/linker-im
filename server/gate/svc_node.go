package gate

import (
    "github.com/Sunmxt/linker-im/server"
    sc "github.com/Sunmxt/linker-im/server/svc/client"
    "github.com/Sunmxt/linker-im/utils/pool"
    "github.com/Sunmxt/linker-im/log"
    "net"
    "net/rpc"
    "hash/fnv"
    "encoding/binary"
    "strings"
    "sync/atomic"
)

const (
    NODE_AVALIABLE = uint8(0)
    NODE_UNAVALIABLE = uint8(1)
)

type ServiceNodeEvent struct {
    Node *ServiceNode
    NewState uint8
    OldState uint8
}

type ServiceNode struct {
    Name    string
    State   uint8
    clients *pool.Pool
    id      server.NodeID
    addr    string
    rpcPath string
    hash    uint32
}

func OpenServiceNode(id server.NodeID, name, address, rpcPath string, maxConcurrentRequest, maxConnection int) *ServiceNode {
    n := ServiceNode{
        Name, name,
        id: id,
        addr: address,
        rpcPath: rpcPath,
        State: NODE_UNAVALIABLE,
    }
    n.ResetHash()
    n.clients = pool.NewPool(n, maxConnection, maxConcurrentRequest)
    return n
}

func connectServiceClient(helper func() (*pool.Drip, error)) (*sc.ServiceClient, error) {
    drip, err := helper()
    if err != nil {
        return nil, err
    }
    wrap, ok := drip.X.(*sc.ServiceClient)
    if !ok {
        err = errors.New("Invalid node connection type from pool.")
        log.Error(err.Error())
        return nil, err
    }
    wrap.Extra = drip
    return wrap, nil
}

func (n *ServiceNode) Connect(timeout uint32) (*sc.ServiceClient, error) {
    return connectServiceClient(func () (*pool.Drip, error) {
        return n.pool.Get(true, timeout)
    })
}

func (n *ServiceNode) TryConnect() (*sc.ServiceClient, error) {
    return connectServiceClient(func () (*pool.Drip, error) {
        return n.pool.Get(false, 0)
    })
}

func (n *ServiceNode) Disconnect(conn *sc.ServiceClient, err error) {
    drip, ok := conn.Extra.(*pool.Drip)
    if !ok {
        log.Panic("Broken extra field of ServiceClient.")
    }
    drip.Release(err)
}

func (n *ServiceNode) Keepalive(event chan *ServiceNodeStateEvent) error {
    conn, err, old, state := n.Connect(), n.State, NODE_UNAVALIABLE
    defer func() {
        if event != nil {
            event <- &ServiceNodeStateEvent{
                Node: n,
                OldState: old,
                NewState: state,
            }
        }
        n.State = state
        conn.Disconnect(client, err)
    }()
    if err != nil {
        return err
    }
    if _, err = conn.Echo("ping"); err != nil {
        return err
    }
    state = NODE_AVALIABLE
    return nil
}

// Pool interfaces
func (n *ServiceNode) Healthy(x interfaces{}, err error) bool {
    netErr, isNetErr := err.(net.Error)
	return !(err == rpc.ErrShutdown || (isNetErr && !netErr.Timeout()))
}

func (n *ServiceNode) New() (interface{}, error) {
    client, err := rpc.DialHTTPPath("tcp", n.addr, n.rpcPath)
    if err != nil {
        log.Info0("Failed to connect service endpoint \"" + n.Name + "\": " +  err.Error())
        return nil, err
    }
    return &sc.ServiceClient{
        Client: client,
    }, nil
}

func (n *ServiceNode) Destroy(x interface{}) {
    wrap, ok := x.(*sc.ServiceClient)
    if !ok {
        log.Fatalf("Try to destroy object with unexcepted type. (%v)", x)
    }
    wrap.Client.Close()
    log.Info0("Service RPC client closed.")
}

func (n *ServiceNode) Notify(ctx *pool.NotifyContext) {
    switch ctx.Event {
    case pool.POOL_NEW:
        log.Info0("Pool created for service node \"" + ctx.Name + "\".")
    case pool.POOL_NEW_DRIP:
        log.Infof0("New connection for service node \"" + ctx.Name + "\". [dripCount = %v]", ctx.DripCount)
    case pool.POOL_DESTROY_DRIP:
        log.Infof0("Close connection of service node \"" + ctx.Name + "\". [dripCount = %v]", ctx.DripCount)
    case pool.POOL_REMOVE_DRIP:
        log.Infof0("Remove connection from pool of service node \"" + ctx.Name + "\". [dripCount = %v]", ctx.DripCount)
    case pool.POOL_NEW_DRIP_FAILURE:
        log.Infof0("Failure occurs when connect to service node \"" + ctx.Name + "\", [dripCoun = %v]", ctx.DripCount)
    }

	log.DebugLazy(func() string { return ctx.String() })
	log.DebugLazy(func() string { return fmt.Sprintf("pool:%v", ep.clients) })
}

// Hash
func (n *ServiceNode) Rehash() {
    buf := make([]byte, binary.MaxVarintLen32)
	binary.LittleEndian.PutUint32(buf, n.hash)
	fnvHash := fnv.New32a()
	fnvHash.Write(buf)
	n.hash = fnvHash.Sum32()
}

func (n *ServiceNode) ResetHash() {
    fnvHash := fnv.New32a()
	fnvHash.Write([]byte(n.id[:]))
	n.hash = fnvHash.Sum32()
}

func (n *ServiceNode) Hash() uint32 {
    return n.hash
}

func (n *ServiceNode) OrderLess(bucket server.Bucket) bool {
    return strings.Compare(n.Name, bucket.(*ServiceNode).Name) < 0
}
