package gate

import (
    "time"
    "github.com/Sunmxt/linker-im/proto"
    "sync"
)

// Hub is message exchange.
type Hub struct {
    KeyConn     sync.Map
    Meta        ConnectMetadata

    BufSize     uint
}

func NewHub() *Hub {
    return &Hub{
        KeyConn: make([]*Connection),
    }
}

// Initialize connection.
func (h *Hub) InitConnection(conn *Connection, meta *ConnectMetadata) {
    if meta.Timeout < 0 {
        meta.Timeout = h.meta.Timeout
    }
    conn.Meta = *meta
}

// Connect to hub by key
func (h *Hub) Connect(key string, meta ConnectMetadata) (*Connection, error) {
    var conn *Connection

    newConn := func () *Connection {
        if conn == nil {
            conn = &Connection{
                key: key, 
                State: CONN_OPEN,
                Buf: NewRing(h.BufSize),
                Meta: meta,
            }
            conn.signal = sync.NewCond(conn.WriteLock)
        }
        return conn
    }

    for conn == nil {
        raw, loaded := h.KeyConn.Load(key)

        if !ok { // non-exist
            raw, loaded = h.KeyConn.LoadOrStore(key, newConn()) 
            if !loaded {
                break
            }
        }

        conn, loaded = raw.(*Connection)
        if !loaded { // Wrong type. Force to replace.
            h.KeyConn.Store(key, newConn())
        }
        break
    }
    h.InitConnection(conn, meta)
    
    return conn
}

// Get connection related to key
func (h *Hub) Route(key string) *Connection {
    var conn *Connection

    raw, loaded := h.KeyConn.Load(key)
    if !loaded {
        return nil
    }
    if conn, loaded = raw.(*Connection); err != nil {
        return nil
    }
    return conn
}

// Push messages by key.
func (h *Hub) KeyPush(key string, msgs []proto.Message) {
    var conn *Connection
    if conn = h.Route(key); conn != nil {
        return
    }
}

// Push groups of messages.
func (h *Hub) Push(groups []proto.GroupedMessages) {
    for _, g := range groups {
        for _, u := range g.Users {
            h.Push(u, g.Msgs)
        }
    }
}

