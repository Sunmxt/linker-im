package svc

import (
	"fmt"
	"github.com/Sunmxt/linker-im/log"
	"github.com/Sunmxt/linker-im/proto"
	"github.com/Sunmxt/linker-im/server"
)

type ServiceRPC struct {
	server.NodeID
}

func (svc ServiceRPC) Keepalive(gateInfo *proto.KeepaliveGatewayInformation, serviceInfo *proto.KeepaliveServiceInformation) error {
	log.Infof2("Keepalive from gateway %v.", gateInfo.NodeID.String())
	*serviceInfo = proto.KeepaliveServiceInformation{
		NodeID: svc.NodeID,
	}
	return nil
}

func (svc ServiceRPC) PushMessage(msg *proto.MessagePushArguments, reply *proto.MessagePushResult) error {
	return fmt.Errorf("Message pushing not avaliable.")
}
