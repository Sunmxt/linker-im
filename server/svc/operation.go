package svc

import (
	"github.com/Sunmxt/linker-im/log"
	"github.com/Sunmxt/linker-im/proto"
	"github.com/Sunmxt/linker-im/server"
	sc "github.com/Sunmxt/linker-im/server/gate/client"
	"github.com/gomodule/redigo/redis"
	"time"
)

type GateMessageBuffer struct {
	msgs []*proto.MessageBody
}

func (s *Service) pushBulk(namespace string, msgs []proto.Message, result []proto.PushResult) {
	kBuf := make(map[string][]*proto.Message)
	for idx := range msgs {
		keyBufPut(kBuf, msgs[idx].MessageBody.Group, &msgs[idx], 1)
	}
	kErr := make(map[string]error, len(kBuf))
	for group, buf := range kBuf {
		kErr[group] = s.pushGroup(namespace, group, buf)
	}
	for idx := range msgs {
		if err := kErr[msgs[idx].MessageBody.Group]; err != nil {
			result[idx].Msg = err.Error()
		}
	}
}

func (s *Service) pushGroup(namespace, group string, msgs []*proto.Message) error {
	keys, err := s.Model.GetSubscription(namespace, group)
	if err != nil {
		return err
	}
	conn := s.Redis.Get()
	defer conn.Close()
	for idx := range keys {
		keys[idx] = namespace + "." + keys[idx]
		if err = conn.Send("HGET", s.Config.RedisPrefix.Value+"{clientinfo-"+keys[idx]+"}", "gate"); err != nil {
			return err
		}
	}
	if err = conn.Flush(); err != nil {
		return err
	}
	for range keys {
		gate, err := redis.String(conn.Receive())
		if err != nil {
			if err != redis.ErrNil {
				return err
			}
			continue
		}
		go s.pushGate(namespace, gate, keys, msgs)
	}
	return nil
}

func (s *Service) pushGate(namespace, gate string, keys []string, msgs []*proto.Message) {
	raw, loaded := s.gateBuf.Load(gate)
	if !loaded {
		raw, _ = s.gateBuf.LoadOrStore(gate, NewConcurrencyMessageGroupBuffer(uint(len(msgs))))
	}
	buf := raw.(*ConcurrencyMessageGroupBuffer)
	buf.Lock()
	defer buf.Unlock()
	buf.Buf = append(buf.Buf, proto.MessageGroup{
		Keys: keys,
		Msgs: msgs,
	})
	if buf.Flusher > 0 {
		return
	}
	buf.Flusher = 1
	go s.flushGateBuf(gate, buf, 50)
}

func (s *Service) flushGateBuf(gate string, buf *ConcurrencyMessageGroupBuffer, delay uint) {
	time.Sleep(time.Duration(int64(time.Millisecond) * int64(delay)))
	raw, ok := s.gateNode.Load(gate)
	if !ok {
		s.gateBuf.Delete(gate)
		log.Warn("flushGateBuf(): Unknown gate \"" + gate + "\"")
		return
	}
	clientRaw, err := raw.(*server.RPCNode).Connect(0)
	if err != nil {
		log.Error("flushGateBuf(): " + err.Error())
		return
	}
	buf.Lock()
	defer buf.Unlock()
	if err = (*sc.GateClient)(clientRaw).Push(buf.Buf); err != nil {
		log.Warn("flushGateBuf() push error: " + err.Error())
	}
	buf.Buf = buf.Buf[0:0]
	buf.Flusher = 0
}
