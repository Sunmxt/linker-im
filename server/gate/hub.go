package gate

import (
	"github.com/Sunmxt/linker-im/proto"
	"sync"
	"sync/atomic"
	"time"
)

const HUB_RING_DEFAULT_BUFFER_SIZE = 1024

// Hub is message exchange.
type Hub struct {
	KeyConn   sync.Map
	ConnCount int32

	Meta    ConnectMetadata
	BufSize uint

	sigRoute chan *Connection
}

func NewHub(meta ConnectMetadata, bufSize uint) *Hub {
	if bufSize < 2 {
		bufSize = HUB_RING_DEFAULT_BUFFER_SIZE
	}
	return &Hub{
		Meta:     meta,
		sigRoute: make(chan *Connection),
		BufSize:  bufSize,
	}
}

// Initialize connection.
func (h *Hub) InitConnection(conn *Connection, meta *ConnectMetadata) {
	if meta.Timeout < 0 {
		meta.Timeout = h.Meta.Timeout
	}
	conn.Meta = *meta
	conn.State = CONN_CONNECTED
}

// Clean timeout connections
func (h *Hub) Clean(notAfter time.Time) {
	h.Visit(func(key string, conn *Connection) bool {
		if conn.Expire.After(notAfter) {
			h.KeyConn.Delete(key)
		}
		return true
	})
}

func (h *Hub) Visit(fn func(key string, conn *Connection) bool) {
	var count int
	h.KeyConn.Range(func(k, v interface{}) bool {
		var conn *Connection
		key, ok := k.(string)
		if !ok {
			h.KeyConn.Delete(k)
			return true
		}
		conn, ok = v.(*Connection)
		if !ok || conn == nil {
			h.KeyConn.Delete(k)
			return true
		}
		count++
		return fn(key, conn)
	})
	h.ConnCount = int32(count)
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
	meta := ActiveMetadata{}
	h.Visit(func(key string, conn *Connection) bool {
		meta.Proto = conn.Meta.Proto
		meta.Remote = conn.Meta.Remote
		meta.Key = key
		buf = append(buf, meta)
		return true
	})
	return buf
}

// Connect to hub by key
func (h *Hub) Connect(key string, meta ConnectMetadata) (*Connection, error) {
	var conn *Connection

	newConn := func() *Connection {
		if conn == nil {
			conn = &Connection{
				key:   key,
				State: CONN_OPEN,
				Buf:   NewRing(uint64(h.BufSize)),
				Meta:  meta,
			}
			conn.signal = sync.NewCond(&conn.WriteLock)
		}
		return conn
	}

	for conn == nil {
		raw, loaded := h.KeyConn.Load(key)

		if !loaded { // non-exist
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
	h.InitConnection(conn, &meta)

	h.sigRoute <- conn

	return conn, nil
}

// Get connection related to key
func (h *Hub) Route(key string) *Connection {
	var conn *Connection

	raw, loaded := h.KeyConn.Load(key)
	if !loaded {
		return nil
	}
	if conn, loaded = raw.(*Connection); !loaded {
		return nil
	}
	if conn.State != CONN_CONNECTED {
		return nil
	}
	return conn
}

// Push messages by key.
func (h *Hub) KeyPush(key string, msgs []*proto.Message) (uint, uint) {
	var conn *Connection
	if conn = h.Route(key); conn == nil {
		return 0, 0
	}
	return conn.Push(msgs)
}

// Push groups of messages.
func (h *Hub) Push(groups []proto.MessageGroup) error {
	for _, g := range groups {
		for _, key := range g.Keys {
			h.KeyPush(key, g.Msgs)
		}
	}
	return nil
}
