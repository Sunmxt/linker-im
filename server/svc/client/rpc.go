package client

import (
	"errors"
	"github.com/Sunmxt/linker-im/proto"
	"github.com/Sunmxt/linker-im/server"
)

type ServiceClient server.RPCClient

func (c *ServiceClient) Echo(echo string) (string, error) {
	var reply string
	if err := c.Client.Call("ServiceRPC.Echo", &echo, &reply); err != nil {
		return "", err
	}
	return reply, nil
}

func (c *ServiceClient) Push(namespace, session string, msgs []*proto.MessageBody) ([]proto.PushResult, error) {
	reply := proto.MessagePushResult{}
	if err := c.Client.Call("ServiceRPC.Push", &proto.RawMessagePushArguments{
		Msgs:      msgs,
		Session:   session,
		Namespace: namespace,
	}, &reply); err != nil {
		return nil, err
	}
	if reply.Replies == nil {
		reply.Replies = make([]proto.PushResult, 0)
	}
	return reply.Replies, nil
}

func (c *ServiceClient) listEntity(namespace, session string, entityType uint8) ([]string, error) {
	reply := proto.EntityListReply{}
	if err := c.Client.Call("ServiceRPC.EntityList", proto.EntityListArguments{
		Type:      entityType,
		Namespace: namespace,
		Session:   session,
	}, &reply); err != nil {
		return nil, err
	}
	if reply.Msg != "" {
		return nil, errors.New(reply.Msg)
	}
	if reply.Entities == nil {
		reply.Entities = make([]string, 0)
	}
	return reply.Entities, nil
}

func (c *ServiceClient) ListNamespace(session string) ([]string, error) {
	return c.listEntity("", session, proto.ENTITY_NAMESPACE)
}

func (c *ServiceClient) ListGroup(namespace, session string) ([]string, error) {
	return c.listEntity(namespace, session, proto.ENTITY_GROUP)
}

func (c *ServiceClient) ListUser(namespace, session string) ([]string, error) {
	return c.listEntity(namespace, session, proto.ENTITY_USER)
}

func (c *ServiceClient) AlterEntity(req *proto.EntityAlterV1) error {
	var msg string
	if err := c.Client.Call("ServiceRPC.EntityAlter", req, &msg); err != nil {
		return err
	}
	if msg != "" {
		return errors.New(msg)
	}
	return nil
}

func (c *ServiceClient) Subscribe(sub *proto.Subscription) error {
	var msg string
	if err := c.Client.Call("ServiceRPC.Subscribe", sub, &msg); err != nil {
		return err
	}
	if msg != "" {
		return errors.New(msg)
	}
	return nil
}

func (c *ServiceClient) Connect(args *proto.ConnectV1) (*proto.ConnectResultV1, error) {
	reply := proto.ConnectResultV1{}
	if err := c.Client.Call("ServiceRPC.Connect", args, &reply); err != nil {
		return nil, err
	}
	if reply.AuthError != "" {
		return nil, server.NewAuthError(errors.New(reply.AuthError))
	}
	return &reply, nil
}
