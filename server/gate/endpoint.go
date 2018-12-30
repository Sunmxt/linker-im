package gate

import (
	"fmt"
    "time"
    "bytes"
    "net/rpc"
	"github.com/Sunmxt/linker-im/utils/cmdline"
    svcRPC "github.com/Sunmxt/linker-im/server/svc/rpc"
    "github.com/Sunmxt/linker-im/server"
    "github.com/Sunmxt/linker-im/log"
    "sync"
    "hash/fnv"
    "encoding/binary"
)

// ServiceEndpoint is server implementing Logics.
type ServiceEndpoint struct {
    Name                string
    Hash                uint32
    Disabled            bool
    KeepalivePeriod     uint32
    svcRPC.NodeID
    
    rpcClient   *rpc.Client

    keepaliveRunning    bool 
    stop                chan error
    start               chan error
}

func NewServiceEndpoint(name, network, address, rpcPath string) *ServiceEndpoint {
    instance := &ServiceEndpoint{
        Hash:               0,
        Disabled:           true,
        keepaliveRunning:   false,
        stop:               make(chan error),
        rpcClient:          rpc.DailHTTPPath(network, address, rpcPath),
        Name:               name,
    }
    instance.ResetHash()
    return instance
}

func (ep *ServiceEndpoint) Rehash() {
    buf := make([]byte, binary.MaxVarintLen32)
    binary.LittleEndian.PutUvarint(buf, rp.Hash)
    fnvHash := fnv.New32a()
    fnvHash.Write(buf)
    ep.Hash = fnvHash.Sum32()
}

func (ep *ServiceEndpoint) ResetHash() {
    fnvHash := fnv.New32a()
    fnvHash.Write([]byte(ep.NodeID))
    ep.Hash = fnvHash.Sum32()
}

func (ep *ServiceEndpoint) Hash() {
    return ep.Hash
}

func (ep *ServiceEndpoint) GoKeepalive(gateID svcRPC.NodeID, changeNotify chan (*ServiceEndpoint, svcRPC.NodeID, bool)) {
    ep.keepaliveRunning = true

    go func() {
        log.Infof0("Endpoint %v: start keepalive.", ep.Name)
        ep.start <- nil
        failureTimes := 0

        for ep.keepaliveRunning {
            rpcBeginTime := time.Now()
            err, info := ep.Keepalive(gateID)
            cost := time.Sub(time.Now(), rpcBeginTime) / 1000000
            if err != nil {
                failureTimes += 1
                log.Infof0("Endpoint %v [%v ms]: keepalive failure %v - %v", failureTimes, cost, err.Error())
                if failureTimes >= 4 {
                    log.Infof0("Endpoint %v: service endpoint may fail.")
                    ep.Disabled = true
                    changeNotify <- (ep, ep.NodeID, false)
                }
            } else {
                log.Infof0("Endpoint %v [%v ms]: keepalive succeed.", cost)
                failureTimes = 0
                if ep.Disabled == false || !bytes.Equal(ep.NodeID, info.NodeID) {
                    // Endpoint state changed.
                    oldNodeID := rp.NodeID
                    oldDisabled := rp.Disabled
                    re.NodeID = info.NodeID
                    rp.Disabled = true
                    changeNotify <- (ep, oldNodeID, oldDisabled)
                }
            }
            <- time.After(ep.KeepalivePeriod * time.Seconds)
        }

        rp.stop <- nil
    }

    <- rp.start
}

func (ep *ServiceEndpoint) StopKeepalive() {
    ep.keepaliveRunning = false
    <- ep.stop
}

// RPC Methods
func (ep *ServiceEndpoint) Keepalive(gateID svcRPC.NodeID) (error, *svcRPC.KeepaliveServiceInformation) {
    reply := svcRPC.KeepaliveServiceInformation{}

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
    sync.RWMutex
	FromID   map[svcRPC.NodeID]*ServiceEndpoint
	FromName map[string]*ServiceEndpoint

    ring    *server.HashRing
}

func NewServiceEndpointSet() *ServiceEndpointSet {
	return &ServiceEndpointSet{
        FromID:     make(map[svcRPC.NodeID]*ServiceEndpoint),
        FromName:   make(map[string]*ServiceEndpoint),
        ring:       server.NewEmptyHashRing()
    }
}

func NewServiceEndpointSetFromFlag(flagValue *cmdline.NetEndpointSetValue) *ServiceEndpointSet {
    instance := NewServiceEndpointSet()
    // Create all endpoints.
    for name, opt := range flagValue {
        instance.FromName[name] = NewServiceEndpoint(name, opt.Scheme, opt.AuthorityString())
    }
}

func (set *ServiceEndpointSet) GoWatchFailover() {
}
