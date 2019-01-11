package proto

import (
    "github.com/Sunmxt/linker-im/server"
)

const RPC_PATH = "/__rpc_linker_svc"
const RPC_DEBUG_PATH = "/__rpc_linker_svc_debug"

// Dummy
type Dummy struct {}

// Keepalive
type KeepaliveGatewayInformation struct {
	server.NodeID
}

type KeepaliveServiceInformation struct {
	server.NodeID
}

// Push message.

type MessagePushArguments struct {
	Messages []RawMessage
}

type MessagePushResult struct {
    Replies []struct {
        Identifiers MessageIdentifier
        Code    uint8
    }
}

// Namespace
type NamespaceArguments struct {
    Names []string
}

type NamespaceListReply struct {
    Names []string
}
