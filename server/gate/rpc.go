package gate

import (
	"errors"
	"github.com/Sunmxt/linker-im/proto"
	"github.com/Sunmxt/linker-im/server/resource"
	"github.com/Sunmxt/linker-im/utils/pool"
	"net/rpc"
	"sync/atomic"
)

// Errors
var ErrInvalidResourceType = errors.New("Invalid resource type given.")
var ErrTimeout = pool.ErrWaitTimeout

type RPCConnection interface {
	Get() *rpc.Client
	Close(error)
}

type PooledRPCConnection struct {
	closed uint32
	drip   *pool.Drip
	Client *rpc.Client
}

func (conn *PooledRPCConnection) Get() *rpc.Client {
	return conn.Client
}

func (conn *PooledRPCConnection) Close(err error) {
	if !atomic.CompareAndSwapUint32(&conn.closed, 1, 0) {
		return
	}

	conn.Client = nil
	conn.drip.Release(err)
}

func NewServiceRPCClient(timeout uint32) (*ServiceRPCClient, error) {
	rawRes, err := resource.Registry.AuthAccess("svc-endpoint", nil)
	if err != nil {
		return nil, err
	}

	eps, ok := rawRes.(*ServiceEndpointSet)
	if !ok {
		return nil, ErrInvalidResourceType
	}

	drip, err := eps.Get(timeout)
	if err != nil {
		return nil, err
	}

	client, ok := drip.X.(*rpc.Client)
	if err != nil {
		return nil, err
	}

	return &ServiceRPCClient{
		conn: &PooledRPCConnection{
			drip:   drip,
			Client: client,
		},
	}, nil
}

func TryNewServiceRPCClient() (*ServiceRPCClient, error) {
	rawRes, err := resource.Registry.AuthAccess("svc-endpoint", nil)
	if err != nil {
		return nil, err
	}

	eps, ok := rawRes.(*ServiceEndpointSet)
	if !ok {
		return nil, ErrInvalidResourceType
	}
	drip, err := eps.TryGet()
	if err != nil {
		return nil, err
	}

	client, ok := drip.X.(*rpc.Client)
	if err != nil {
		return nil, err
	}

	return &ServiceRPCClient{
		conn: &PooledRPCConnection{
			drip:   drip,
			Client: client,
		},
	}, nil
}

type ServiceRPCClient struct {
	conn    RPCConnection
	lastErr error
}

func (c *ServiceRPCClient) Close(err error) {
	if err == nil {
		err = c.lastErr
	}

	c.conn.Close(err)
}

func (c *ServiceRPCClient) NamespaceAdd(ns []string) error {
	var reply proto.Dummy

	client := c.conn.Get()
	if client == nil {
		return rpc.ErrShutdown
	}

	err := client.Call("ServiceRPC.NamespaceAdd", &proto.NamespaceArguments{
		Names: ns,
	}, &reply)

	if err != nil {
		c.lastErr = err
	}

	return err
}

func (c *ServiceRPCClient) NamespaceList() ([]string, error) {
	var reply proto.NamespaceListReply

	client := c.conn.Get()
	if client == nil {
		return nil, rpc.ErrShutdown
	}

	err := client.Call("ServiceRPC.NamespaceList", &proto.Dummy{}, &reply)
	if err != nil {
		c.lastErr = err
	}

	return reply.Names, err
}

func (c *ServiceRPCClient) NamespaceRemove(ns []string) error {
	var reply proto.Dummy

	client := c.conn.Get()
	if client == nil {
		return rpc.ErrShutdown
	}

	err := client.Call("ServiceRPC.NamespaceRemove", &proto.NamespaceArguments{
		Names: ns,
	}, &reply)

	if err != nil {
		c.lastErr = err
	}

	return err
}
