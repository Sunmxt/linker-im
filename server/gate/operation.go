package gate

import (
	"errors"
	"github.com/Sunmxt/linker-im/log"
	"github.com/Sunmxt/linker-im/proto"
	"github.com/Sunmxt/linker-im/server"
	sc "github.com/Sunmxt/linker-im/server/svc/client"
	"sync"
)

type MessageBucket struct {
	slot   []*proto.MessageBody
	ptr    int
	result []proto.PushResult
}

func (g *Gate) push(session string, msgs []proto.MessageBody) ([]*proto.PushResult, error) {
	// Dispatch
	buckets := make(map[uint32]*MessageBucket)
	for idx := range msgs {
		hash := HashMessage(&msgs[idx])
		bucket, ok := buckets[hash.Hash()]
		if !ok {
			bucket = &MessageBucket{
				slot: make([]*proto.MessageBody, 0, 1),
			}
			buckets[hash.Hash()] = bucket
		}
		bucket.slot = append(bucket.slot, &msgs[idx])
	}
	// Push
	var wg sync.WaitGroup
	for hash, bucket := range buckets {
		node, err := g.LB.HashValueSelect(hash)
		if err != nil {
			return nil, err
		}
		wg.Add(1)
		go g.bucketPush(&wg, node, bucket, session, 0)
	}
	wg.Wait()

	// Serialize
	result := make([]*proto.PushResult, len(msgs))
	for idx := range msgs {
		bucket := buckets[HashMessage(&msgs[idx]).Hash()]
		if bucket.result == nil {
			log.Warn("Gate.push() nil push result.")
			continue
		}
		if bucket.ptr >= len(bucket.result) {
			log.Warnf("Gate.push() push result too short. (%v/%v)", bucket.ptr, len(bucket.result))
			continue
		}
		result[idx] = &bucket.result[bucket.ptr]
		bucket.ptr++
	}
	return result, nil
}

func (g *Gate) randomPush(session string, msgs []proto.MessageBody) ([]proto.MessageIdentifier, error) {
	return nil, errors.New("Not implemented.")
}

func (g *Gate) bucketPush(wg *sync.WaitGroup, node *server.RPCNode, bucket *MessageBucket, session string, connTimeout int) error {
	defer wg.Done()
	client, err := node.Connect(0)
	if err != nil {
		bucket.result = make([]proto.PushResult, len(bucket.slot))
		for idx := range bucket.result {
			bucket.result[idx].Msg = err.Error()
		}
		log.Error("bucketPush connection failure: " + err.Error())
		return err
	}

	if bucket.result, err = (*sc.ServiceClient)(client).Push(session, bucket.slot); err != nil {
		log.Error("bucketPush RPC failure: " + err.Error())
	}
	node.Disconnect(client, err)
	return err
}
