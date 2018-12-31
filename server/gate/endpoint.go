package gate

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"github.com/Sunmxt/linker-im/log"
	"github.com/Sunmxt/linker-im/server"
	svcRPC "github.com/Sunmxt/linker-im/server/svc/rpc"
	"github.com/Sunmxt/linker-im/utils/cmdline"
	"hash/fnv"
	"net/rpc"
	"sync"
	"sync/atomic"
	"time"
)

// ServiceEndpoint is server implementing Logics.
type ServiceEndpoint struct {
	Name            string
	Disabled        bool
	KeepalivePeriod uint32
	svcRPC.NodeID

	rpcClient *rpc.Client

	hash             uint32
	keepaliveRunning uint32
	stop             chan error
	start            chan error
}

func NewServiceEndpoint(name, network, address, rpcPath string) (*ServiceEndpoint, error) {
	client, err := rpc.DialHTTPPath(network, address, rpcPath)
	if err != nil {
		return nil, err
	}

	instance := &ServiceEndpoint{
		hash:             0,
		Disabled:         true,
		keepaliveRunning: 0,
		stop:             make(chan error),
		start:            make(chan error),
		rpcClient:        client,
		Name:             name,
	}
	instance.ResetHash()
	return instance, nil
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

func (ep *ServiceEndpoint) disable(set *ServiceEndpointSet) {
	set.lock.Lock()
	defer set.lock.Unlock()

	// remove myself from ring
	set.ring.RemoveHash(ep.hash)
	ep.Disabled = true
	log.Infof0("Disable inactive service endpoint %v.", ep.Name)
}

func (ep *ServiceEndpoint) enable(set *ServiceEndpointSet) {
	set.lock.Lock()
	defer set.lock.Unlock()

	// append myself to ring
	set.ring.Append(ep)
	ep.Disabled = false
	log.Infof0("Enable active service endpoint %v.", ep.Name)
}

func (ep *ServiceEndpoint) changeID(nodeID *svcRPC.NodeID, set *ServiceEndpointSet) {
	if bytes.Equal(nodeID[:], ep.NodeID[:]) {
		return
	}

	set.lock.Lock()
	defer set.lock.Unlock()

	if !bytes.Equal(ep.NodeID[:], svcRPC.EMPTY_NODE_ID) {
		delete(set.FromID, ep.NodeID.AsKey())
		log.Infof0("Service endpoint %v ID Changed: %v -> %v", ep.NodeID.String(), nodeID.String())
	} else {
		log.Infof0("Service endpoint %v joined with ID: %v", nodeID.String())
	}
	set.FromID[nodeID.AsKey()] = ep
	ep.NodeID.Assign(nodeID)
}

func (ep *ServiceEndpoint) GoKeepalive(set *ServiceEndpointSet) error {
	if !atomic.CompareAndSwapUint32(&ep.keepaliveRunning, 0, 1) {
		return nil
	}

	go func() {
		log.Infof0("Endpoint %v: start keepalive.", ep.Name)
		ep.start <- nil
		failureTimes := 0

		for ep.keepaliveRunning > 0 {
			rpcBeginTime := time.Now()
			err, info := ep.Keepalive(set.GateID)
			cost := time.Now().Sub(rpcBeginTime) / 1000000
			if err != nil {
				failureTimes += 1
				log.Infof0("Endpoint %v [%v ms]: keepalive failure %v - %v", failureTimes, cost, err.Error())
				if failureTimes >= 4 {
					log.Infof0("Endpoint %v: service endpoint may fail.")
					ep.disable(set)
				}
			} else {
				log.Infof0("Endpoint %v [%v ms]: keepalive succeed.", cost)
				failureTimes = 0
				ep.changeID(&info.NodeID, set)
				if ep.Disabled == true {
					ep.enable(set)
				}
			}
			<-time.After(time.Duration(ep.KeepalivePeriod) * time.Second)
		}

		ep.keepaliveRunning = 0
		ep.stop <- nil
	}()

	return <-ep.start
}

func (ep *ServiceEndpoint) StopKeepalive() error {
	atomic.SwapUint32(&ep.keepaliveRunning, 0)
	return <-ep.stop
}

// RPC Methods
func (ep *ServiceEndpoint) Keepalive(gateID svcRPC.NodeID) (error, *svcRPC.KeepaliveServiceInformation) {
	reply := &svcRPC.KeepaliveServiceInformation{}

	err := ep.rpcClient.Call("ServiceRPC.Keepalive", &svcRPC.KeepaliveGatewayInformation{
		NodeID: gateID,
	}, reply)
	if err != nil {
		return err, nil
	}

	return nil, reply
}

// ServiceEndpointSet resources
type ServiceEndpointSet struct {
	lock sync.RWMutex

	FromID   map[string]*ServiceEndpoint
	FromName map[string]*ServiceEndpoint
	GateID   svcRPC.NodeID

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

func NewServiceEndpointSetFromFlag(flagValue *cmdline.NetEndpointSetValue) *ServiceEndpointSet {
	var err error
	instance := NewServiceEndpointSet()
	// Create all endpoints.
	for name, opt := range flagValue.Endpoints {
		instance.FromName[name], err = NewServiceEndpoint(name, opt.Scheme, opt.AuthorityString(), svcRPC.RPC_PATH)
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
		err = endpoint.GoKeepalive(set)
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

func (set *ServiceEndpointSet) GoKeepalive() error {
	if !atomic.CompareAndSwapUint32(&set.keepaliveRunning, 0, 1) {
		return nil
	}
	set.lock.RLock()
	defer set.lock.RUnlock()

	var err error = nil
	for name, endpoint := range set.FromName {
		err = endpoint.GoKeepalive(set)
		if err != nil {
			log.Infof0("Failed to start keepalive routine for endpoint %v: %v", name, err.Error())
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