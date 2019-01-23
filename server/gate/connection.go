package gate

import (
    "github.com/Sunmxt/linker-im/proto"
    "time"
    "sync/atomic"
)

const (
    PROTO_HTTP = iota
    PROTO_TCP 
    PROTO_WEBSOCKET
)

type ConnectMetadata struct {
    Proto       uint
    Remote      string
    Timeout     int
}

const (
    CONN_OPEN   = iota
    CONN_CONNECTED
    CONN_CLOSE
)

type Connection struct {
    key     string

    State   uint8
    Expire  time.Time
    Meta    ConnectMetadata

    Buf         *Ring
    bulk        int

    signal      *sync.Cond
    WriteLock   sync.Mutex
    ReadLock    sync.Mutex
}

func (c *Connection) consume(buf []proto.Message, max int) ([]proto.Message, int) {
    var count int = 0
    c.ReadLock.Lock()
    defer c.ReadLock.Unlock()
    readc := len(buf)
    for count = 0; max < 1 || count < max ; count++ {
        msg := c.Buf.Read()
        if msg != nil {
            break
        }
        buf = append(buf, *c.Buf.Read())
    }
    return buf, count
}

func (c *Connection) wait(wake chan struct{}) {
    c.signal.Wait()
    c.WriteLock.Unlock()
    wake <- chan struct{}{}
}

func (c *Connection) Receive(buf []proto.Message, max int, bulk int, timeout int) uint {
    if bulk < 1 {
        bulk = 1
    }
    if max > 0 && bulk > max {
        bulk = max
    }

    var (
        readc, cnt uint = 0, 0
    )
    notAfter, writeLocked, wake := time.Now(), false, make(chan struct{}, 1)
    buf = buf[0:0]

    if timeout > 0 {
        notAfter = notAfter.Add(time.Duration(int64(timeout) * time.Millisecond))
        go func() {
            time.Sleep(time.Duration(int64(timeout) * time.Millisecond)
            wake <- struct{}{}
        }
    }

    for {
        if c.Buf.Count() > 0 { // consume if any message
            buf, cnt = c.consume(buf, max)
            continue

        } else if writeLocked { // wait
            c.bulk = bulk - readc
            writeLocked = false
            go wait(wake)
            <-wake
            continue
        }
        
        now := time.Now()
        if (timeout >= 0 && now.After(notAfter)) || readc >= bulk {
            break
        }

        if !writeLocked { // prepare waiting.
            c.WriteLock.Lock()
            writeLocked = true
            // then try entering waiting state.
        }
    }

    if writeLocked { // edge case: reach exit condition after waiting prepared.
        c.WriteLock.Unlock()
    }

    return readc
}
