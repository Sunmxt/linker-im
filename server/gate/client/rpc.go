package client

import (
	"github.com/Sunmxt/linker-im/proto"
	"github.com/Sunmxt/linker-im/server"
	"net/rpc"
)

type GateClient server.RPCClient

func (c *GateClient) Push(session string, msgs []proto.MessageGroup) error {
	if err := c.Client.Call("GateRPC.Push", &proto.MessageGroup{
		Gups: msgs,
	}, &struct{}{}); err != nil {
		return err
	}
	return nil
}
