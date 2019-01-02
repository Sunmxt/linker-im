package proto

import (
    "github.com/Sunmxt/linker-im/server"
)

const RPC_PATH = "/__rpc_linker_svc"
const RPC_DEBUG_PATH = "/__rpc_linker_svc_debug"

// Keepalive
type KeepaliveGatewayInformation struct {
	server.NodeID
}

type KeepaliveServiceInformation struct {
	server.NodeID
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
