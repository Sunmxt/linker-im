package svc

import (
	"github.com/Sunmxt/linker-im/proto"
	"sync"
	"time"
)

type TimeSerializer struct {
	lock     sync.Mutex
	sequence uint32
	stamp    uint64
}

func (s *TimeSerializer) Get(count uint32) (uint64, uint32) {
	s.lock.Lock()
	defer s.lock.Unlock()
	stamp := uint64(time.Now().Unix())
	if stamp != s.stamp {
		s.stamp = stamp
		s.sequence = 1
	}
	s.sequence += count
	return s.stamp, s.sequence
}

func (s *TimeSerializer) SerializeMessage(user string, msgs []*proto.MessageBody, result []proto.PushResult) error {
	// Serialize
	total := uint32(len(result))
	stamp, seq := s.Get(total)
	seq -= total
	for idx := range result {
		msgs[idx].User = user
		result[idx].MessageIdentifier.Timestamp = stamp
		result[idx].MessageIdentifier.Sequence = seq + uint32(idx)
	}
	return nil
}
