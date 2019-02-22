package server

import (
	"encoding/binary"
	"errors"
	"fmt"
	"github.com/Sunmxt/linker-im/log"
	"github.com/Sunmxt/linker-im/utils/pool"
	guuid "github.com/satori/go.uuid"
	"hash/fnv"
	"net/rpc"
	"strings"
)

// Errors
var ErrInvalidNodeIDString = errors.New("Invalid ID string.")

// Node ID
type NodeID guuid.UUID

var EMPTY_NODE_ID []byte = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

func NewNodeID() NodeID {
	return NodeID(guuid.NewV4())
}

func (n *NodeID) String() string {
	return strings.Replace(guuid.UUID(*n).String(), "-", "", -1)
}

func (n *NodeID) AsKey() string {
	return string(n[:])
}

func (n *NodeID) Assign(id *NodeID) {
	copy(n[:], id[:])
}

func (n *NodeID) FromString(raw string) error {
	return (*guuid.UUID)(n).UnmarshalText([]byte(raw))
}

// RPC Node
type RPCClient struct {
	Extra interface{}
	*rpc.Client
}

const (
	NODE_AVALIABLE   = uint8(0)
	NODE_UNAVALIABLE = uint8(1)
)

type RPCNodeEvent struct {
	Node     *RPCNode
	NewState uint8
	OldState uint8
}

type RPCDripInterface interface {
	pool.DripInterface
	Keepalive(client *RPCClient, event chan *RPCNodeEvent) error
}

type RPCNode struct {
	Name    string
	State   uint8
	clients *pool.Pool
	id      NodeID
	hash    uint32
	ifce    RPCDripInterface
}

func OpenRPCNode(id NodeID, name string, ifce RPCDripInterface, maxConcurrentRequest, maxConnection int) *RPCNode {
	n := &RPCNode{
		Name:  name,
		id:    id,
		State: NODE_UNAVALIABLE,
		ifce:  ifce,
	}
	n.ResetHash()
	n.clients = pool.NewPool(n, maxConnection, maxConcurrentRequest)
	return n
}

func connectRPCClient(helper func() (*pool.Drip, error)) (*RPCClient, error) {
	drip, err := helper()
	if err != nil {
		return nil, err
	}
	wrap, ok := drip.X.(*RPCClient)
	if !ok {
		err = errors.New("Not a RPCClient from pool.")
		log.Error(err.Error())
		return nil, err
	}
	wrap.Extra = drip
	return wrap, nil
}

func (n *RPCNode) Connect(timeout uint32) (*RPCClient, error) {
	return connectRPCClient(func() (*pool.Drip, error) {
		return n.clients.Get(true, timeout)
	})
}

func (n *RPCNode) TryConnect() (*RPCClient, error) {
	return connectRPCClient(func() (*pool.Drip, error) {
		return n.clients.Get(false, 0)
	})
}

func (n *RPCNode) Disconnect(conn *RPCClient, err error) {
	drip, ok := conn.Extra.(*pool.Drip)
	if !ok {
		log.Panic("Broken extra field of RPCClient.")
	}
	drip.Release(err)
}

func (n *RPCNode) Close() {
	n.clients.Close()
}

func (n *RPCNode) Keepalive(event chan *RPCNodeEvent) error {
	old, state := n.State, NODE_UNAVALIABLE
	conn, err := n.Connect(0)
	defer func() {
		if event != nil {
			event <- &RPCNodeEvent{
				Node:     n,
				OldState: old,
				NewState: state,
			}
		}
		n.State = state
		n.Disconnect(conn, err)
	}()
	if err != nil {
		return err
	}
	if err = n.ifce.Keepalive(conn, event); err != nil {
		return err
	}
	state = NODE_AVALIABLE
	return nil
}

// Drip interface
func (n *RPCNode) Healthy(x interface{}, err error) bool {
	return n.ifce.Healthy(x, err)
}

func (n *RPCNode) New() (interface{}, error) {
	raw, err := n.ifce.New()
	if err != nil {
		return nil, err
	}
	client, ok := raw.(*rpc.Client)
	if !ok {
		return nil, errors.New("RPCDripInterface.New() return invalid RPC client.")
	}
	return &RPCClient{
		Client: client,
	}, nil
}

func (n *RPCNode) Destroy(x interface{}) {
	wrap, ok := x.(*RPCClient)
	if !ok {
		log.Panic("RPCNode.Destroy(): Broken pool of RPCNode. Invalid value type returned.", x)
	}
	n.ifce.Destroy(wrap.Client)
}

func (n *RPCNode) Notify(ctx *pool.NotifyContext) {
	n.ifce.Notify(ctx)
	log.DebugLazy(func() string { return fmt.Sprintf("pool:%v", n.clients) })
}

// Hash
func (n *RPCNode) Rehash() {
	buf := make([]byte, binary.MaxVarintLen32)
	binary.LittleEndian.PutUint32(buf, n.hash)
	fnvHash := fnv.New32a()
	fnvHash.Write(buf)
	n.hash = fnvHash.Sum32()
}

func (n *RPCNode) ResetHash() {
	fnvHash := fnv.New32a()
	fnvHash.Write([]byte(n.id[:]))
	n.hash = fnvHash.Sum32()
}

func (n *RPCNode) Hash() uint32 {
	return n.hash
}

func (n *RPCNode) OrderLess(bucket Bucket) bool {
	return strings.Compare(n.Name, bucket.(*RPCNode).Name) < 0
}
