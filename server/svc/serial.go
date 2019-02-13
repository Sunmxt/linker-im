package svc

import (
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
