package proto

import (
	"github.com/Sunmxt/linker-im/server"
)

const RPC_PATH = "/__rpc_linker_svc"
const RPC_DEBUG_PATH = "/__rpc_linker_svc_debug"

// Dummy
type Dummy struct{}

// Keepalive
type KeepaliveGatewayInformation struct {
	server.NodeID
}

type KeepaliveServiceInformation struct {
	server.NodeID
}

// Push message.
type MessageGroup struct {
	Msgs  []Message
	Users []string
}

type MessagePushArguments struct {
	Gups []MessageGroup
}

type MessagePushResult struct {
	Replies []struct {
		Timestamp uint64
		Sequence  uint64
		Code      uint8
	}
}

type Subscription struct {
	Group    string
	NotAfter int64
}

type SubscribeArguments struct {
	User      string
	Namespace string
	Subs      []Subscription
}

type NamespaceOperationArguments struct {
	Names []string
}

type NamespaceListReply struct {
	Names  []string
	ErrMsg string
}
