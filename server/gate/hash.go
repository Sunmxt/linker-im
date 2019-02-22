package gate

import (
	"github.com/Sunmxt/linker-im/proto"
	"github.com/Sunmxt/linker-im/server"
	"hash/fnv"
)

type Hashed uint32

func (h Hashed) Hash() uint32 { return uint32(h) }

func HashMessage(msg *proto.MessageBody) server.Hashable {
	fnvHash := fnv.New32a()
	fnvHash.Write([]byte(msg.Group))
	return Hashed(fnvHash.Sum32())
}
