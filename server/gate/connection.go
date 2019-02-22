package gate

import (
	ilog "github.com/Sunmxt/linker-im/log"
	"github.com/Sunmxt/linker-im/proto"
	"sync"
	"time"
)

const (
	PROTO_HTTP = iota
	PROTO_TCP
	PROTO_WEBSOCKET
)

type ConnectMetadata struct {
	Proto   uint
	Remote  string
	Timeout int
}

type ActiveMetadata struct {
	Proto  uint
	Remote string
	Key    string
}

const (
	CONN_OPEN = iota
	CONN_CONNECTED
	CONN_CLOSE
)

type Connection struct {
	key string

	State  uint8
	Expire time.Time
	Meta   ConnectMetadata

	Buf  *Ring
	bulk int

	signal    *sync.Cond
	WriteLock sync.Mutex
	ReadLock  sync.Mutex
}

func (c *Connection) wait(wake chan struct{}) {
	c.signal.Wait()
	c.WriteLock.Unlock()
	wake <- struct{}{}
}

func (c *Connection) Push(msgs []*proto.Message) (uint, uint) {
	if len(msgs) < 1 {
		return 0, 0
	}

	var overc, readLocked uint
	var idx int
	c.WriteLock.Lock()
	defer c.WriteLock.Unlock()

	for idx := range msgs {
		override, err := c.Buf.Write(msgs[idx], readLocked > 0)
		if override {
			ilog.Warnf("Drop message for full ring buffer.")
		}
		if err == ErrRingFull {
			c.ReadLock.Lock()
			readLocked = 1
			continue
		}
		if readLocked > 0 {
			overc++
		}
		idx++
	}
	if readLocked > 0 {
		c.ReadLock.Unlock()
	}
	if c.bulk < 0 || c.Buf.Count() >= uint64(c.bulk) {
		c.signal.Broadcast()
	}

	return uint(idx), overc
}

func (c *Connection) consume(buf []proto.Message, max int) ([]proto.Message, int) {
	var count int = 0
	c.ReadLock.Lock()
	defer c.ReadLock.Unlock()
	for count = 0; max < 1 || count < max; count++ {
		msg := c.Buf.Read()
		if msg == nil {
			break
		}
		buf = append(buf, *msg)
	}
	return buf, count
}

func (c *Connection) Receive(buf []proto.Message, max int, bulk int, timeout int) []proto.Message {
	if bulk < 1 {
		bulk = 1
	}
	if max > 0 && bulk > max {
		bulk = max
	}

	var cnt int = 0
	notAfter, writeLocked, wake := time.Now(), false, make(chan struct{}, 1)
	buf = buf[0:0]

	if timeout > 0 {
		notAfter = notAfter.Add(time.Duration(timeout) * time.Millisecond)
		go func() {
			time.Sleep(time.Duration(timeout) * time.Millisecond)
			wake <- struct{}{}
		}()
	}

	for {
		if c.Buf.Count() > 0 { // consume if any message
			buf, cnt = c.consume(buf, max)
			bulk -= cnt
			max -= cnt
			continue

		} else if writeLocked { // wait
			c.bulk = bulk
			writeLocked = false
			go c.wait(wake)
			<-wake
			continue
		}

		now := time.Now()
		if (timeout >= 0 && now.After(notAfter)) || bulk <= 0 {
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

	return buf
}
