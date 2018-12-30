package rpc

import (
	"fmt"
	"github.com/Sunmxt/linker-im/log"
	"github.com/Sunmxt/linker-im/server"
	guuid "github.com/satori/go.uuid"
	"strings"
)

// Node ID
type NodeID guuid.UUID

func NewNodeID() NodeID {
	return NodeID(guuid.NewV4())
}

func (n *NodeID) String() string {
	return strings.Replace(guuid.UUID(*n).String(), "-", "", -1)
}

// RPC Runtime
const RPC_PREFIX = "/_rpc"
const _RPC_PATH = "/__linker_svc"
const _RPC_DEBUG_PATH = "/__linker_svc_debug"
const RPC_PATH = RPC_PREFIX + _RPC_PATH
const RPC_DEBUG_PATH = RPC_PREFIX + _RPC_DEBUG_PATH

type ServiceRPC struct {
	NodeID
}

// Keepalive
type KeepaliveGatewayInfomation struct {
	NodeID
}

type KeepaliveServiceInformation struct {
	NodeID
}

func (svc ServiceRPC) Keepalive(gateInfo *KeepaliveGatewayInfomation, serviceInfo *KeepaliveServiceInformation) error {
	log.Infof0("Keepalive from gateway %v.", gateInfo.NodeID.String())
	*serviceInfo = KeepaliveServiceInformation{
		NodeID: svc.NodeID,
	}
	return nil
}

// Push message.
type MessagePushArguments struct {
	server.Message
	Namespace string
}

type MessagePushResult struct {
	Timestamp  uint64
	SequenceID uint32
}

func (svc ServiceRPC) PushMessage(msg *MessagePushArguments, reply *MessagePushResult) error {
	return fmt.Errorf("Message pushing not avaliable.")
}
