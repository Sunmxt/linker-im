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

func (g *Gate) push(namespace, session string, msgs []proto.MessageBody) ([]*proto.PushResult, error) {
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
		go g.bucketPush(&wg, node, bucket, session, namespace, 0)
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

func (g *Gate) randomPush(namespace, session string, msgs []proto.MessageBody) ([]proto.MessageIdentifier, error) {
	return nil, errors.New("Not implemented.")
}

func (g *Gate) bucketPush(wg *sync.WaitGroup, node *server.RPCNode, bucket *MessageBucket, session, namespace string, connTimeout int) error {
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

	if bucket.result, err = (*sc.ServiceClient)(client).Push(namespace, session, bucket.slot); err != nil {
		log.Error("bucketPush RPC failure: " + err.Error())
	}
	node.Disconnect(client, err)
	return err
}

func (g *Gate) roundRobinDo(do func(*sc.ServiceClient) error) error {
	var client *server.RPCClient
	node, err := g.LB.RoundRobinSelect()
	if err != nil {
		return err
	}
	if client, err = node.Connect(0); err != nil {
		return err
	}
	err = do((*sc.ServiceClient)(client))
	node.Disconnect(client, err)
	return err
}

func (g *Gate) subscribe(sub proto.Subscription) error {
	return g.roundRobinDo(func(client *sc.ServiceClient) error {
		return client.Subscribe(&sub)
	})
}

func (g *Gate) connect(conn *proto.ConnectV1) (*proto.ConnectResultV1, error) {
	var reply *proto.ConnectResultV1
	var err error
	g.roundRobinDo(func(client *sc.ServiceClient) error {
		reply, err = client.Connect(conn)
		return err
	})
	if err != nil {
		return reply, err
	}
	if reply.Key == "" {
		reply.Key = reply.Session
	}
	gate.KeySession.Store(conn.Namespace+"."+reply.Session, reply.Key)
	return reply, nil
}

func (g *Gate) hubConnect(namespace, session string, meta ConnectMetadata) (*Connection, error) {
	key := g.sessionKey(namespace, session)
	if key == "" {
		return nil, server.NewAuthError(errors.New("Connection rejected."))
	}
	return g.Hub.Connect(key, meta)
}
