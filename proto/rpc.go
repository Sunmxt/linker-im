package proto

const RPC_PATH = "/__rpc_linker_svc"
const RPC_DEBUG_PATH = "/__rpc_linker_svc_debug"

// push message group.
type MessageGroup struct {
	Msgs    []Message
	Session []string
}

type MessagePushArguments struct {
	Gups []MessageGroup
}

// push raw message.
type RawMessagePushArguments struct {
	Msgs    []*MessageBody
	Session string
}

type PushResult struct {
	MessageIdentifier
	Msg string `json:"m"`
}
type MessagePushResult struct {
	Replies []PushResult
}

type Subscription struct {
	Namespace string
	User      string
	Group     string
}

const (
	ENTITY_NAMESPACE = uint8(1)
	ENTITY_USER      = uint8(2)
	ENTITY_GROUP     = uint8(3)

	ENTITY_ADD = uint8(1)
	ENTITY_DEL = uint8(2)
)

type EntityAlterArguments struct {
	Namespace string
	Entities  []string
	Operation uint8
	Type      uint8
}

type EntityListArguments struct {
	Namespace string
	Type      uint8
}

type EntityListReply struct {
	Entities []string
	Msg      string
}
