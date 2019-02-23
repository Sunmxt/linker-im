package proto

const RPC_PATH = "/__rpc_linker_svc"
const RPC_DEBUG_PATH = "/__rpc_linker_svc_debug"

// push message group.
type MessageGroup struct {
	Msgs []*Message
	Keys []string
}

type MessagePushArguments struct {
	Gups []MessageGroup
}

// push raw message.
type RawMessagePushArguments struct {
	Msgs      []*MessageBody
	Session   string
	Namespace string
}

type PushResult struct {
	MessageIdentifier
	Msg string `json:"m,omitempty"`
}

type MessagePushResult struct {
	Replies     []PushResult
	IsAuthError bool
	Msg         string
}

type EntityListArguments struct {
	Namespace string
	Session   string
	Type      uint8
}

type EntityListReply struct {
	Entities []string
	Msg      string
}
