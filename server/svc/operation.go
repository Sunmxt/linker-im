package svc

import (
	"github.com/Sunmxt/linker-im/proto"
)

func (s *Service) push(key string, msgs []*proto.MessageBody) ([]proto.PushResult, error) {
	// Serialize
	total := uint32(len(msgs))
	result := make([]proto.PushResult, 0, total)
	stamp, seq := s.serial.Get(total)
	seq -= total
	for idx := range msgs {
		msgs[idx].User = key
		result = append(result, proto.PushResult{
			MessageIdentifier: proto.MessageIdentifier{
				Timestamp: stamp,
				Sequence:  seq + uint32(idx),
			},
		})
	}
	return result, nil
}
