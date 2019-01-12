package gate

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/Sunmxt/linker-im/log"
	"github.com/Sunmxt/linker-im/proto"
	"github.com/Sunmxt/linker-im/server"
	"github.com/Sunmxt/linker-im/utils/cmdline"
	"github.com/Sunmxt/linker-im/utils/pool"
	"hash/fnv"
	"net"
	"net/rpc"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

type ServiceEndpoint struct {
	Name     string
	Network  string
	Address  string
	RPCPath  string
	Disabled bool
	server.NodeID

	GateID server.NodeID

	clients *pool.Pool

	hash             uint32
	keepaliveRunning uint32

	stop     chan error
	start    chan error
	changeID chan server.NodeID
}

func NewServiceEndpoint(name, network, address, rpcPath string, maxConcurrentRequest, maxConnection int) (*ServiceEndpoint, error) {
	instance := &ServiceEndpoint{
		hash:             0,
		Disabled:         true,
		keepaliveRunning: 0,
		stop:             make(chan error),
		start:            make(chan error),
		changeID:         make(chan server.NodeID, 0),
		Name:             name,
		Network:          network,
		Address:          address,
		RPCPath:          rpcPath,
	}
	instance.ResetHash()
	instance.clients = pool.NewPool(instance, maxConnection, maxConcurrentRequest)
	return instance, nil
}

func (ep *ServiceEndpoint) New() (interface{}, error) {
	client, err := rpc.DialHTTPPath(ep.Network, ep.Address, ep.RPCPath)
	if err != nil {
		log.Infof0("Failed to connect service endpoint \"%v\" (%v).", ep.Name, err.Error())
		return nil, err
	}
	return client, nil
}

func (ep *ServiceEndpoint) Destroy(x interface{}) {
	client, ok := x.(*rpc.Client)
	if !ok {
		log.Fatalf("Try to destroy object with unexcepted type. (%v)", x)
		return
	}

	client.Close()
	log.Infof0("Service RPC client closed.")
}

func (ep *ServiceEndpoint) Healthy(x interface{}, err error) bool {
	netErr, isNetErr := err.(net.Error)
	return !(err == rpc.ErrShutdown || (isNetErr && !netErr.Timeout()))
}

func (ep *ServiceEndpoint) bootstrap() {
	log.Infof0("Bootstraping connection of endpoint \"%v\"", ep.Name)

	for {
		drip, err := ep.clients.Get(false)
		if err == nil && drip != nil {
			drip.Release(nil)
			break
		}

		log.Infof0("Bootstraping failure of endpoint \"%v\" (%v).", ep.Name, err.Error())
		<-time.After(time.Duration(10) * time.Second)
	}
}

func (ep *ServiceEndpoint) Notify(ctx *pool.NotifyContext) {
	var nodeID server.NodeID

	switch ctx.Event {
	case pool.POOL_NEW:
		// bootstrap endpoint connection.
		go ep.bootstrap()

	case pool.POOL_NEW_DRIP:
		log.Infof0("Service RPC client added. [dripCount = %v]", ctx.DripCount)

		drip := ctx.Get()
		if drip == nil {
			log.Warn("Keepalive routine cannot start (nil drip returned). service endpoint may not be used by load balancer.")
			break
		}
		ep.changeID <- ep.NodeID
		go ep.keepalive(drip)

	case pool.POOL_DESTROY_DRIP:
		log.Infof0("Service RPC client destroy. [dripCount = %v]", ctx.DripCount)

	case pool.POOL_REMOVE_DRIP:
		log.Infof0("Service RPC client removed. [dripCount = %v]", ctx.DripCount)
		if ctx.DripCount == 0 {
			copy(nodeID[:], server.EMPTY_NODE_ID[:])
			ep.changeID <- nodeID

			// reboot endpoint connection.
			go ep.bootstrap()
		}

	case pool.POOL_NEW_DRIP_FAILURE:
		log.Infof0("Failed to add Service RPC client. (%v) [dripCount = %v]", ctx.Error.Error(), ctx.DripCount)
	}

	log.DebugLazy(func() string { return ctx.String() })
	log.DebugLazy(func() string { return fmt.Sprintf("pool:%v", ep.clients) })
}

func (ep *ServiceEndpoint) Rehash() {
	buf := make([]byte, binary.MaxVarintLen32)
	binary.LittleEndian.PutUint32(buf, ep.hash)
	fnvHash := fnv.New32a()
	fnvHash.Write(buf)
	ep.hash = fnvHash.Sum32()
}

func (ep *ServiceEndpoint) ResetHash() {
	fnvHash := fnv.New32a()
	fnvHash.Write([]byte(ep.NodeID[:]))
	ep.hash = fnvHash.Sum32()
}

func (ep *ServiceEndpoint) Hash() uint32 {
	return ep.hash
}

func (ep *ServiceEndpoint) OrderLess(bucket server.Bucket) bool {
	return strings.Compare(ep.Name, bucket.(*ServiceEndpoint).Name) < 0
}

func (ep *ServiceEndpoint) keepalive(drip *pool.Drip) {
	var err error

	log.Infof0("Start keepalive with endpoint \"%v\".", ep.Name)

	client, ok := drip.X.(*rpc.Client)
	if !ok {
		log.Warn("Keepalive routine of endpoint \"%v\"cannot start (not a rpc client). service endpoint may not be used by load balancer.", ep.Name)
		return
	}
	for failTime := 1; ep.keepaliveRunning > 0; {
		reply := &proto.KeepaliveServiceInformation{}

		rpcBeginTime := time.Now()
		err = client.Call("ServiceRPC.Keepalive", &proto.KeepaliveServiceInformation{
			NodeID: ep.GateID,
		}, reply)
		cost := int64(time.Now().Sub(rpcBeginTime)) / 1000000

		if err != nil {
			if !ep.Healthy(client, err) {
				log.Infof0("Keepalive check of endpoint \"%v\" failure (%v).", ep.Name, err.Error())
				break
			} else {
				failTime++
				log.Infof0("[%v ms] Keepalive failure %v with service endpoint \"%v\". (%v)", cost, failTime, ep.Name, err.Error())
				if failTime > 2 {
					log.Infof0("Service endpoint \"%v\" may fail.", ep.Name)
					break
				}
			}
		} else {
			if failTime > 0 {
				log.Infof0("[%v ms] Keepalive succeed with service endpoint \"%v\".", cost, ep.Name)
			}

			failTime = 0
			ep.changeID <- reply.NodeID
		}
	}

	drip.Release(err)
	log.Infof0("Keepalive routine of endpoint \"%v\" exiting...", ep.Name)
}

func (ep *ServiceEndpoint) ringUpdate(gateID server.NodeID, set *ServiceEndpointSet) {
	log.Infof0("Start watching connection state of endpoint \"%v\".", ep.Name)

	ep.start <- nil
	ep.GateID = gateID

	for count := 0; ep.keepaliveRunning > 0; count++ {
		nodeID, more := <-ep.changeID
		if !more {
			break
		}
		log.Debugf("Receive NodeID \"%v\" of endpoint \"%v\".", nodeID.String(), ep.Name)

		if !bytes.Equal(nodeID[:], ep.NodeID[:]) {
			// ID Changed.
			set.lock.Lock()

			if !bytes.Equal(ep.NodeID[:], server.EMPTY_NODE_ID) {
				// remove myself from ring
				delete(set.FromID, ep.NodeID.AsKey())
				set.ring.RemoveHash(ep.hash)
				ep.ResetHash()

				log.Infof0("Service endpoint \"%v\" ID changed from \"%v\" to \"%v\"", ep.Name, ep.NodeID.String(), nodeID.String())
			} else {
				log.Infof0("Service endpoint \"%v\" joined with ID \"%v\"", ep.Name, nodeID.String())
			}

			if !bytes.Equal(nodeID[:], server.EMPTY_NODE_ID) {
				// append myself to ring
				set.FromID[nodeID.AsKey()] = ep
				set.ring.Append(ep)
			} else {
				log.Infof0("Disable inactive service endpoint \"%v\".", ep.Name)
			}

			ep.NodeID.Assign(&nodeID)
			set.lock.Unlock()

			log.Debugf("HashRing: %v", set.ring)
		}

		if count >= ep.clients.Len()+1 {
			<-time.After(time.Duration(set.KeepalivePeriod) * time.Second)
			count = 0
		}
	}

	ep.stop <- nil
}

func (ep *ServiceEndpoint) StartKeepalive(set *ServiceEndpointSet) error {
	if !atomic.CompareAndSwapUint32(&ep.keepaliveRunning, 0, 1) {
		return nil
	}

	go ep.ringUpdate(set.GateID, set)

	return <-ep.start
}

func (ep *ServiceEndpoint) StopKeepalive() error {
	atomic.SwapUint32(&ep.keepaliveRunning, 0)
	close(ep.changeID)

	ep.clients.Close()

	return <-ep.stop
}

// ServiceEndpointSet resources
type ServiceEndpointSet struct {
	lock sync.RWMutex

	FromID          map[string]*ServiceEndpoint
	FromName        map[string]*ServiceEndpoint
	GateID          server.NodeID
	KeepalivePeriod uint

	ring             *server.HashRing
	keepaliveRunning uint32
}

func NewServiceEndpointSet() *ServiceEndpointSet {
	return &ServiceEndpointSet{
		FromID:           make(map[string]*ServiceEndpoint),
		FromName:         make(map[string]*ServiceEndpoint),
		ring:             server.NewEmptyHashRing(),
		keepaliveRunning: 0,
	}
}

func NewServiceEndpointSetFromFlag(flagValue *cmdline.NetEndpointSetValue, maxConn, maxCurrency int) *ServiceEndpointSet {
	var err error
	instance := NewServiceEndpointSet()
	// Create all endpoints.
	for name, opt := range flagValue.Endpoints {
		instance.FromName[name], err = NewServiceEndpoint(name, opt.Scheme, opt.AuthorityString(), proto.RPC_PATH, maxCurrency, maxConn)
		if err != nil {
			log.Errorf("Cannot create ServiceEndpoint: %v", name)
		}
	}
	return instance
}

func (set *ServiceEndpointSet) AddEndpoint(endpoint *ServiceEndpoint) error {
	set.lock.Lock()
	defer set.lock.Unlock()

	var err error = nil

	if _, exists := set.FromName[endpoint.Name]; exists {
		return fmt.Errorf("Endpoint %v already exists.", endpoint.Name)
	}
	set.FromName[endpoint.Name] = endpoint

	if set.keepaliveRunning > 0 {
		err = endpoint.StartKeepalive(set)
	}
	if err != nil {
		delete(set.FromName, endpoint.Name)
	}

	return err
}

func (set *ServiceEndpointSet) RemoveEndpoint(name string) (*ServiceEndpoint, error) {
	set.lock.Lock()
	defer set.lock.Unlock()

	var err error = nil
	endpoint, exists := set.FromName[name]
	if !exists {
		return nil, fmt.Errorf("Endpoint %v not exists.", name)
	}
	if endpoint.keepaliveRunning > 0 {
		err = endpoint.StopKeepalive()
	}
	delete(set.FromName, name)
	if _, exists := set.FromID[endpoint.NodeID.AsKey()]; exists {
		delete(set.FromID, endpoint.NodeID.AsKey())
	}

	return endpoint, err
}

func (set *ServiceEndpointSet) GoKeepalive(nodeID server.NodeID, period uint) error {
	if !atomic.CompareAndSwapUint32(&set.keepaliveRunning, 0, 1) {
		return nil
	}

	set.GateID, set.KeepalivePeriod = nodeID, period
	set.lock.RLock()
	defer set.lock.RUnlock()

	var err error = nil
	for name, endpoint := range set.FromName {
		err = endpoint.StartKeepalive(set)
		if err != nil {
			log.Infof0("Failed to start keepalive routine of endpoint \"%v\" (\"%v\")", name, err.Error())
			break
		}
	}
	if err != nil {
		for _, endpoint := range set.FromName {
			if endpoint.keepaliveRunning > 0 {
				endpoint.StopKeepalive()
			}
		}
	}
	return err
}

func (set *ServiceEndpointSet) StopKeepalive() {
	set.lock.RLock()
	defer set.lock.RUnlock()

	for name, endpoint := range set.FromName {
		if endpoint.keepaliveRunning > 0 {
			err := endpoint.StopKeepalive()
			log.Errorf("Error occurs when stop keepalive routine of endpoint %v: %v", name, err.Error())
		}
	}

	atomic.SwapUint32(&set.keepaliveRunning, 0)
}
