package svc

import (
	"github.com/Sunmxt/linker-im/proto"
	"sync"
)

type ConcurrencyMessageGroupBuffer struct {
	lock    sync.RWMutex
	Buf     []proto.MessageGroup
	Flusher int
}

func NewConcurrencyMessageGroupBuffer(capacity uint) *ConcurrencyMessageGroupBuffer {
	return &ConcurrencyMessageGroupBuffer{
		Buf: make([]proto.MessageGroup, 0, capacity),
	}
}

func (b *ConcurrencyMessageGroupBuffer) Lock()    { b.lock.Lock() }
func (b *ConcurrencyMessageGroupBuffer) RLock()   { b.lock.RLock() }
func (b *ConcurrencyMessageGroupBuffer) Unlock()  { b.lock.Unlock() }
func (b *ConcurrencyMessageGroupBuffer) RUnlock() { b.lock.RUnlock() }

func keyBufPutMany(kb map[string][]*proto.Message, key string, msgs []*proto.Message, capacity int) {
	buf, ok := kb[key]
	if !ok {
		buf = make([]*proto.Message, 0, capacity)
	}
	kb[key] = append(buf, msgs...)
}

func keyBufPut(kb map[string][]*proto.Message, key string, msg *proto.Message, capacity int) {
	keyBufPutMany(kb, key, []*proto.Message{msg}, capacity)
}
