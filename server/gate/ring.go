package gate

import (
	"errors"
	"github.com/Sunmxt/linker-im/proto"
)

var ErrRingFull = errors.New("Ring is full.")

type Ring struct {
	buf    []proto.Message
	readc  uint64
	writec uint64
	mask   uint64
}

func NewRing(size uint64) *Ring {
	for {
		t := size & (size - 1)
		if t == 0 {
			break
		}
		size = t
	}
	size <<= 1
	ring := &Ring{
		buf:    make([]proto.Message, size, size),
		readc:  0,
		writec: 0,
		mask:   size - 1,
	}
	return ring
}

func (r *Ring) Write(msg *proto.Message, override bool) (bool, error) {
	if r.writec-r.readc > r.mask {
		if !override {
			return false, ErrRingFull
		}
		r.readc++
	} else {
		override = false
	}
	r.buf[r.writec&r.mask] = *msg
	r.writec++
	return override, nil
}

func (r *Ring) Read() *proto.Message {
	if r.Count() == 0 {
		return nil
	}
	msg := &r.buf[r.readc&r.mask]
	r.readc++
	return msg
}

func (r *Ring) Count() uint64 {
	rc, wc := r.writec, r.readc
	if wc < rc {
		return 0
	}
	return wc - rc
}
