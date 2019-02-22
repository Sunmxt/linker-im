package client

import (
	"github.com/Sunmxt/linker-im/proto"
	"github.com/Sunmxt/linker-im/server"
)

type GateClient server.RPCClient

func (c *GateClient) Echo(echo string) (string, error) {
	var reply string
	err := c.Client.Call("GateRPC.Echo", echo, &reply)
	return reply, err
}

func (c *GateClient) Push(msgs []proto.MessageGroup) error {
	if err := c.Client.Call("GateRPC.Push", &proto.MessagePushArguments{
		Gups: msgs,
	}, &struct{}{}); err != nil {
		return err
	}
	return nil
}
