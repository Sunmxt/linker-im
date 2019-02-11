package gate

import (
	"errors"
	"github.com/Sunmxt/linker-im/log"
	"github.com/Sunmxt/linker-im/server"
	"sync"
	"sync/atomic"
)

var ErrNodeMissing = errors.New("load balancer: No avaliable RPC endpoint.")
var ErrNodeType = errors.New("Wrong type of endpoint.")

type ServiceLB struct {
	lock sync.RWMutex
	ring *server.HashRing

	FromName map[string]*ServiceNode
	round    uint32

	event        chan *ServiceNodeEvent
	running      sync.Map
	runningCount uint32
	stop         chan struct{}
}

func NewLB() *ServiceLB {
	return &ServiceLB{
		ring:     server.NewEmptyHashRing(),
		event:    make(chan *ServiceNodeEvent),
		stop:     make(chan struct{}, 1),
		FromName: make(map[string]*ServiceNode),
	}
}

func (lb *ServiceLB) Node(name string) *ServiceNode {
	lb.lock.RLock()
	defer lb.lock.RUnlock()
	return lb.FromName[name]
}

func (lb *ServiceLB) AddNode(name string, node *ServiceNode) error {
	lb.lock.Lock()
	defer lb.lock.Unlock()
	if n, exists := lb.FromName[name]; exists && n != node {
		return errors.New("Node \"" + name + "\"already exists.")
	}
	lb.FromName[name] = node
	return nil
}

func (lb *ServiceLB) RemoveNode(name string) {
	lb.lock.Lock()
	defer lb.lock.Unlock()
	node, ok := lb.FromName[name]
	if !ok {
		return
	}
	lb.ring.RemoveHash(node.Hash())
	delete(lb.FromName, name)
	go node.Close()
}

func (lb *ServiceLB) Keepalive() {
	lb.lock.RLock()
	defer lb.lock.RUnlock()
	count := 0
	for k, node := range lb.FromName {
		log.Info2("Keepalive routine of node \"" + k + "\" starts.")
		if _, loaded := lb.running.LoadOrStore(node, struct{}{}); !loaded {
			go node.Keepalive(lb.event)
			count++
		}
	}
	atomic.AddUint32(&lb.runningCount, 1) // Count running goroutines
	select {
	case <-lb.stop:
	default:
	}
	go func() {
		for count > 0 {
			event := <-lb.event
			if event.OldState != event.NewState {
				switch event.OldState {
				case NODE_UNAVALIABLE:
					log.Info0("Node \"" + event.Node.Name + "\" becomes avaliable.")
					lb.ring.Append(event.Node)

				case NODE_AVALIABLE:
					log.Info0("Node \"" + event.Node.Name + "\" becomes unavaliable.")
					lb.ring.RemoveHash(event.Node.Hash())
				}
			}
			lb.running.Delete(event.Node)
			count--
		}
		if atomic.AddUint32(&lb.runningCount, uint32(0xFFFFFFFF)) == 0 {
			lb.stop <- struct{}{}
		}
	}()
}

// Close wait until all goroutines exited.
func (lb *ServiceLB) Close() {
	<-lb.stop
}

func (lb *ServiceLB) selectNode(helper func() server.Bucket) (*ServiceNode, error) {
	bucket := helper()
	if bucket == nil {
		return nil, ErrNodeMissing
	}
	node, ok := bucket.(*ServiceNode)
	if !ok {
		return nil, ErrNodeType
	}
	return node, nil
}

func (lb *ServiceLB) HashSelect(h server.Hashable) (*ServiceNode, error) {
	return lb.selectNode(func() server.Bucket {
		lb.lock.RLock()
		defer lb.lock.RUnlock()
		_, bucket := lb.ring.Hit(h)
		return bucket
	})
}

func (lb *ServiceLB) RoundRobinSelect() (*ServiceNode, error) {
	return lb.selectNode(func() server.Bucket {
		round := atomic.AddUint32(&lb.round, 1)
		lb.lock.RLock()
		defer lb.lock.RUnlock()
		if r := lb.ring.Len(); r <= 0 {
			round = 0
		} else {
			round = round % uint32(r)
		}
		return lb.ring.At(int(round))
	})
}
