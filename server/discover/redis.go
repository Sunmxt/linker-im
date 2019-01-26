package discover

import (
    "github.com/gomodule/redigo/redis"
    "sync"
)

type RedisRegistry struct {
    redis   *redis.Pool
    prefix  string

    service sync.Map
    nodes   map[string]*Node
}

func NewRedisPoolRegistry(pool *redis.Pool, prefix string) (Registry, error) {
    return RedisRegistry{
        redis: pool,
        prefix: prefix,
        service: make(map[string]*RedisServiceEntry, 0),
    }, nil
}

func NewRedisRegistry(network, prefix string, maxIdle, maxActive int) (Registry, error) {
    return NewRedisRegistry(&redis.Pool{
        Dail: func() (redis.Conn, error) {
            return redis.Dail("tcp", network)
        },
        MaxIdle: maxIdle,
        MaxActive: maxActive,
        Wait:       true,
        MaxConnLifetime: 0,
        IdleTimeout: 0,
    }, prefix)
}

func Connect(args ...interface{}) (Registry, error) {
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
        return NewRedisPoolRegistry(first, prefix, maxIdle, maxActive)
    }

    return nil, ErrInvalidArguments
}

func (r *RedisRegistry) Service(name string) (Service, error) {
    var svc *RedisServiceEntry

    raw, loaded := r.service.Load(name)
    for {
        if loaded {
            svc, loaded = raw.(*RedisServiceEntry)
        }
        if !loaded {
            raw, loaded = r.service.LoadOrStore(name, NewRedisServiceEntry(name, r))
            if loaded {
                continue
            }
        }
        break
    }

    return svc, nil
}

func (r *RedisRegistry) Poll() (bool, error) {
    conn := redis.Get()
    defer conn.Close()

    svcs, err := redis.Strings(conn.Do("SMEMBERS", r.prefix + "{dig-services}"), nil)
    if err != nil {
        return false, err
    }
    changed := false
    for idx := range svcs {
        entry, ok := r.service[svcs[idx]]
        if !ok {
            r.service[svcs[idx]] = nil
            changed = true
        }
        
        if entry != nil {
            entryChanged, err := entry.poll(conn)
            if err != nil {
                return updated, err
            }
            if entryChanged {
                changed = true
            }
        }
    }

    err = r.resolveNodes(conn)
    return changed, err
}

func (r *RedisRegistry) Node(name string) *Node {
    var node *Node

    raw, loaded := r.nodes.Load(name)
    for {
        if loaded {
            node, _ := raw.(*Node)
        } else {
            raw, loaded = r.nodes.LoadOrStore(name, NewEmptyNode(name))
            if loaded {
                continue
            }
        }
        break
    }

    return node
}

// Visit all nodes focused on.
func (r *RedisRegistry) VisitNodes(fn func (name string, node *Node) bool) {
    var (
        ok bool
        name string
        node *Node
    )
    r.nodes.Range(func (k, v interface{}) bool {
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
        ok bool
        name string
        svc Service
    )
    r.nodes.Range(func (k, v interface{}) bool {
        if name, ok = k.(string); !ok {
            r.nodes.Delete(k)
        }
        if svc, ok = v.(Service); !ok {
            r.nodes.Delete(k)
        }
        return fn(name, svc)
    })
}

func (r *RedisRegistry) resolveNodes(conn redis.Conn) error {
    var (
        err error
        meta    map[string]string
    )
    focusNames := make([]string, 0) // TODO: optimize.
    focusNodes := make([]*Node, 0)
    r.VisitNodes(func (name string, node *Node) bool {
        if err = conn.Send("HGETALL", r.prefix + "{dig-node-\"" + name + "\"}"); err != nil {
            return false
        }
        focusNames = append(focusNames, name)
        focusNodes = append(focusNodes, name)
        return true
    })
    if err != nil {
        return false, err
    }
    if err = conn.Flush(); err != nil {
        return false, err
    }

    for idx := range focusNames {
        if meta, err = StringMap(conn.Receive()); err != nil {
            return updated, err
        }
        node, name := focusNodes[idx], focusNames[name]
        if node == nil {
            node = r.Node(name)
        }
        node.Name = name
        node.Metadata = meta
    }

    return nil
}

func (r *RedisRegistry) Close() {
}

type RedisServiceEntry struct {
    name        string
    nodes       map[string]struct{}
    publish     map[string]*Node
    registry    *RedisRegistry

    lock        sync.Mutex
    sig         *sync.Cond
}

func NewRedisServiceEntry(name string, registry *RedisRegistry) *RedisServiceEntry {
    entry := &RedisServiceEntry{
        name: name,
        node: make(map[string]map[string]string),
        publish: make(map[string]*Node),
        registry: registry,
    }
    entry.sig = sync.NewCond(entry.lock)
    return entry
}

func (s *RedisServiceEntry) poll(conn redis.Conn) (bool, error) {
    updated := false

    nodes, err := redis.Strings(conn.Do("SMEMBERS", r.prefix + "{dig-service-" + s.name + "-node}"))
    if err != nil {
        return false
    }
    for idx := range nodes {
        _, ok := r.nodes[nodes[idx]]
        if !ok {
            updated = true
            r.nodes[nodes[idx]] = struct{}{}
        }
    }
    focusNodes := make([]string, 0, len(s.nodes))
    for name, _ := range s.nodes {
        err = conn.Send("GET", r.prefix + "{dig-service-" + s.name + "-node-" + name "-present}")
        if err != nil {
            return updated, err
        }
        focusNodes = append(focusNodes, name)
    }
    for idx := range focusNodes {
        present, err := redis.Int64(conn.Receive())
        if err != nil {
            if err == redis.ErrNil {
                delete(s.nodes, focusNodes[idx])
                updated = true
            } else {
                return updated, err
            }
        }
    }
    if changed {
        s.sig.Broadcast()
    }
    return updated, nil
}

func (s *RedisServiceEntry) Nodes() []string {
    nodes := make([]string, 0, len(s.nodes))
    for node, _ := s.nodes {
        nodes = append(nodes, node)
    }
    return nodes
}

func (s *RedisServiceEntry) Watch() error {
    s.lock.Lock()
    defer s.lock.Unlock()
    s.sig.Wait()
    return nil
}

func (s *RedisServiceEntry) Publish(node *Node) error {
    s.Registry.redisConnect()
}
