package gate

import (
    "time"
    "github.com/Sunmxt/linker-im/proto"
    "sync"
    "sync/atomic"
)

// Hub is message exchange.
type Hub struct {
    KeyConn     sync.Map
    ConnCount   int32

    Meta        ConnectMetadata

    BufSize     uint
}

// Initialize connection.
func (h *Hub) InitConnection(conn *Connection, meta *ConnectMetadata) {
    if meta.Timeout < 0 {
        meta.Timeout = h.meta.Timeout
    }
    conn.Meta = *meta
    conn.State = CONN_CONNECTED
}


// Clean timeout connections
func (h *Hub) Clean(notAfter time.Time) {
    h.Visit(func (key string, conn *Connection) bool {
        if conn.Expire.After(notAfter) {
            h.KeyConn.Delete(k)
        }
        return true
    })
}

func (h *Hub) Visit(fn func (key string, conn *Connection) bool) {
    var count int
    h.KeyConn.Range(func (k, v interface{}) bool {
        if key, ok := k.(string); !ok {
            h.KeyConn.Delete(k)
            return true
        }
        conn, ok := v.(*Connection)
        if !ok || conn == nil {
            h.KeyConn.Delete(k)
            return true
        }
        count++
        return fn(key, conn)
    })
    h.ConnCount = count
}


// Count return connection count.
func (h *Hub) Count() uint32 {
    cnt := h.ConnCount
    if cnt < 0 {
        return 0
    }
    return uint32(cnt)
}

func (h *Hub) Snapshot(buf []ActiveMetadata) []ActiveMetadata {
    buf = buf[0:0]
    metas, meta := make([]ActiveMetadata, 0, h.Count()), ActiveMetadata{}
    h.Visit(func (key string, conn *Connection) bool {
        meta.Proto = conn.Meta.Proto
        meta.Remote = conn.Meta.Remote
        meta.Key = key
        buf = append(buf, meta)
    })
    return buf
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
                atomic.AddInt32(&h.ConnCount, 1)
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
    if conn != CONN_CONNECTED {
        return nil
    }
    return conn
}

// Push messages by key.
func (h *Hub) KeyPush(key string, msgs []proto.Message) uint {
    var conn *Connection
    if conn = h.Route(key); conn != nil {
        return
    }
    return conn.Push(msgs)
}

// Push groups of messages.
func (h *Hub) Push(groups []proto.GroupedMessages) {
    for _, g := range groups {
        for _, u := range g.Users {
            h.Push(u, g.Msgs)
        }
    }
}

