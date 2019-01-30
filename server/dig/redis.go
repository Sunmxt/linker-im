package dig

import (
	//"fmt"
	"github.com/gomodule/redigo/redis"
	"sync"
)

type RedisConnector struct{}

func (c *RedisConnector) Connect(args ...interface{}) (Registry, error) {
	return RedisDriverConnect(args...)
}

type RedisRegistry struct {
	redis  *redis.Pool
	prefix string

	service sync.Map
	nodes   sync.Map
	publish map[string]*Node
}

func NewRedisPoolRegistry(pool *redis.Pool, prefix string) (Registry, error) {
	return &RedisRegistry{
		redis:   pool,
		prefix:  prefix,
		publish: make(map[string]*Node),
	}, nil
}

func NewRedisRegistry(network, prefix string, maxIdle, maxActive int) (Registry, error) {
	return NewRedisPoolRegistry(&redis.Pool{
		Dial: func() (redis.Conn, error) {
			return redis.Dial("tcp", network)
		},
		MaxIdle:         maxIdle,
		MaxActive:       maxActive,
		Wait:            true,
		MaxConnLifetime: 0,
		IdleTimeout:     0,
	}, prefix)
}

func RedisDriverConnect(args ...interface{}) (Registry, error) {
	if len(args) < 2 {
		return nil, ErrInvalidArguments
	}
	switch first := args[0].(type) {
	case *redis.Pool:
		if len(args) != 2 {
			return nil, ErrInvalidArguments
		}
		prefix, ok := args[1].(string)
		if !ok {
			return nil, ErrInvalidArguments
		}
		return NewRedisPoolRegistry(first, prefix)
	case string:
		var maxIdle, maxActive int
		if len(args) != 4 {
			return nil, ErrInvalidArguments
		}
		prefix, ok := args[1].(string)
		if !ok {
			return nil, ErrInvalidArguments
		}
		if maxIdle, ok = args[2].(int); !ok {
			return nil, ErrInvalidArguments
		}
		if maxActive, ok = args[3].(int); !ok {
			return nil, ErrInvalidArguments
		}
		return NewRedisRegistry(first, prefix, maxIdle, maxActive)
	}

	return nil, ErrInvalidArguments
}

func (r *RedisRegistry) Service(name string) (Service, error) {
	svc, _ := r.GetService(name, true)
	return svc, nil
}

func (r *RedisRegistry) GetService(name string, focus bool) (*RedisServiceEntry, bool) {
	var svc *RedisServiceEntry
	created := true

	raw, loaded := r.service.Load(name)
	for {
		if loaded {
			switch t := raw.(type) {
			case nil:
				if focus { // create Service instance.
					r.service.Delete(name)
					loaded = false
					continue
				}
			case *RedisServiceEntry: // instance already created.
				svc = t
			}
			break
		} else {
			if focus {
				raw, _ = r.service.LoadOrStore(name, NewRedisServiceEntry(name, r))
			} else {
				raw, _ = r.service.LoadOrStore(name, nil)
			}
			created = false
			loaded = true
		}
	}

	return svc, created
}

func (r *RedisRegistry) redisConnect() (redis.Conn, error) {
	pool := r.redis
	if pool == nil {
		return nil, ErrClosed
	}
	return pool.Get(), nil
}

func (r *RedisRegistry) Poll() (bool, error) {
	conn, err := r.redisConnect()
	if err != nil {
		return false, err
	}
	defer conn.Close()
	changed := false

	var svcs []string
	svcs, err = redis.Strings(conn.Do("SMEMBERS", r.prefix+"{dig-services}"))
	if err != nil {
		if err != redis.ErrNil {
			return false, err
		}
	} else {
		for idx := range svcs {
			_, created := r.GetService(svcs[idx], false)
			if !created { // New service discovered.
				changed = true
			}
		}
	}
	r.VisitServices(func(name string, svc Service) bool {
		if svc == nil {
			return true
		}
		entry, ok := svc.(*RedisServiceEntry)
		if !ok {
			return true
		}
		ok, err = entry.poll(conn)
		if err != nil {
			return false
		}
		if ok {
			changed = true
		}
		return true
	})

	if err = r.resolveNodes(conn); err != nil {
		return changed, err
	}

	err = r.publishNodes(conn)

	return changed, err
}

func (r *RedisRegistry) Node(name string) (*Node, error) {
	var node *Node

	raw, loaded := r.nodes.Load(name)
	for {
		if loaded {
			node, _ = raw.(*Node)
		} else {
			raw, loaded = r.nodes.LoadOrStore(name, NewEmptyNode(name))
			if loaded {
				continue
			}
		}
		break
	}

	return node, nil
}

func (r *RedisRegistry) Publish(node *Node) error {
	if node == nil {
		return ErrInvalidArguments
	}
	r.publish[node.Name] = node
	return nil
}

// Visit all nodes focused on.
func (r *RedisRegistry) VisitNodes(fn func(name string, node *Node) bool) {
	var (
		ok   bool
		name string
		node *Node
	)
	r.nodes.Range(func(k, v interface{}) bool {
		if name, ok = k.(string); !ok {
			r.nodes.Delete(k)
		}
		if node, ok = v.(*Node); !ok {
			r.nodes.Delete(k)
		}
		return fn(name, node)
	})
}

// Visit all services focused on.
func (r *RedisRegistry) VisitServices(fn func(name string, svc Service) bool) {
	var (
		ok   bool
		name string
		svc  Service
	)
	r.service.Range(func(k, v interface{}) bool {
		if name, ok = k.(string); !ok {
			r.nodes.Delete(k)
		}
		if svc, ok = v.(Service); !ok && v != nil {
			r.nodes.Delete(k)
		}
		return fn(name, svc)
	})
}

func (r *RedisRegistry) resolveNodes(conn redis.Conn) error {
	var (
		err  error
		meta map[string]string
	)
	focusNames := make([]string, 0) // TODO: optimize.
	focusNodes := make([]*Node, 0)
	r.VisitNodes(func(name string, node *Node) bool {
		if err = conn.Send("HGETALL", r.prefix+"{dig-node-"+name+"}"); err != nil {
			return false
		}
		focusNames = append(focusNames, name)
		focusNodes = append(focusNodes, node)
		return true
	})
	if err != nil {
		return err
	}
	if err = conn.Flush(); err != nil {
		return err
	}

	for idx := range focusNames {
		if meta, err = redis.StringMap(conn.Receive()); err != nil {
			return err
		}
		node, name := focusNodes[idx], focusNames[idx]
		if node == nil {
			node, _ = r.Node(name)
		}
		node.Name = name
		node.Metadata = meta
	}

	return nil
}

func (r *RedisRegistry) publishNodes(conn redis.Conn) error {
	var count uint = 0
	var err error
	for name, node := range r.publish {
		if node != nil && node.Metadata != nil {
			for k, v := range node.Metadata {
				if err = conn.Send("HSET", r.prefix+"{dig-node-"+name+"}", k, v); err != nil {
					return err
				}
				count++
			}
		}
	}
	if err = conn.Flush(); err != nil {
		return err
	}
	for count > 0 {
		if _, err = conn.Receive(); err != nil {
			return err
		}
		count--
	}
	return nil
}

func (r *RedisRegistry) Close() {
	r.redis = nil
}

type RedisServiceEntry struct {
	name     string
	nodes    sync.Map
	publish  map[string]uint
	registry *RedisRegistry

	lock sync.Mutex
	sig  *sync.Cond
}

func NewRedisServiceEntry(name string, registry *RedisRegistry) *RedisServiceEntry {
	entry := &RedisServiceEntry{
		name:     name,
		publish:  make(map[string]uint),
		registry: registry,
	}
	entry.sig = sync.NewCond(&entry.lock)
	return entry
}

func (s *RedisServiceEntry) VisitNodes(fn func(node string) bool) {
	s.nodes.Range(func(k, v interface{}) bool {
		name, ok := k.(string)
		if !ok {
			s.nodes.Delete(k)
		}
		return fn(name)
	})
}

func (s *RedisServiceEntry) poll(conn redis.Conn) (bool, error) {
	updated := false
	var err error

	nodes, err := redis.Strings(conn.Do("SMEMBERS", s.registry.prefix+"{dig-service-"+s.name+"-node}"))

	if err != nil {
		if err != redis.ErrNil {
			return false, err
		}
	} else {
		for idx := range nodes {
			_, ok := s.nodes.Load(nodes[idx])
			if !ok {
				updated = true
				s.nodes.LoadOrStore(nodes[idx], struct{}{})
			}
		}
	}
	focusNodes := make([]string, 0) // TODO: optimize
	s.VisitNodes(func(name string) bool {
		err = conn.Send("GET", s.registry.prefix+"{dig-service-"+s.name+"-node-"+name+"-present}")
		if err != nil {
			return false
		}
		focusNodes = append(focusNodes, name)
		return true
	})
	if err != nil {
		return updated, err
	}
	if err = conn.Flush(); err != nil {
		return updated, err
	}
	count := 0
	for idx := range focusNodes {
		_, err := redis.Int64(conn.Receive())
		if err != nil {
			if err == redis.ErrNil {
				s.nodes.Delete(focusNodes[idx])
				conn.Send("SREM", s.registry.prefix+"{dig-service-"+s.name+"-node}", focusNodes[idx])
				updated = true
			} else {
				return updated, err
			}
		}
	}
	if err = conn.Flush(); err != nil {
		return updated, err
	}
	for count > 0 {
		if _, err = conn.Receive(); err != nil {
			return updated, err
		}
		count--
	}

	// Publish
	focusNodes = focusNodes[0:0]
	for name, timeout := range s.publish {
		focusNodes = append(focusNodes, name)
		if timeout > 0 {
			conn.Send("SET", s.registry.prefix+"{dig-service-"+s.name+"-node-"+name+"-present}", 1, "ex", timeout)
		} else {
			conn.Send("SET", s.registry.prefix+"{dig-service-"+s.name+"-node-"+name+"-present}", 1)
		}
		conn.Send("SADD", s.registry.prefix+"{dig-service-"+s.name+"-node}", name)
	}
	if err = conn.Flush(); err != nil {
		return updated, err
	}
	for range focusNodes {
		if _, err = conn.Receive(); err != nil {
			return updated, err
		}
	}

	if updated {
		s.sig.Broadcast()
	}
	return updated, nil
}

func (s *RedisServiceEntry) Nodes() []string {
	nodes := make([]string, 0) // TODO: optimize
	s.VisitNodes(func(node string) bool {
		nodes = append(nodes, node)
		return true
	})
	return nodes
}

func (s *RedisServiceEntry) Watch() error {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.sig.Wait()
	return nil
}

func (s *RedisServiceEntry) Publish(node *Node) error {
	if node == nil {
		return ErrInvalidArguments
	}
	s.publish[node.Name] = node.Timeout
	return s.registry.Publish(node)
}
