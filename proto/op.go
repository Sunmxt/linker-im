package proto

const (
	CONN_SESSION = uint8(0)
	CONN_BASIC   = uint8(1)

	OP_SUB_ADD    = uint8(0)
	OP_SUB_CANCEL = uint8(1)

	ENTITY_NAMESPACE = uint8(1)
	ENTITY_USER      = uint8(2)
	ENTITY_GROUP     = uint8(3)

	ENTITY_ADD = uint8(1)
	ENTITY_DEL = uint8(2)
)

const (
	OP_DUMMY       = uint16(0)
	OP_SUB         = uint16(1)
	OP_UNSUB       = uint16(2)
	OP_CONNECT     = uint16(3)
	OP_KEEPALIVE   = uint16(4)
	OP_PUSH        = uint16(5)
	OP_PULL        = uint16(6)
	OP_GROUP_ENUM  = uint16(7)
	OP_USER_ENUM   = uint16(8)
	OP_NS_ENUM     = uint16(9)
	OP_GROUP_ALTER = uint16(10)
	OP_USER_ALTER  = uint16(11)
	OP_NS_ALTER    = uint16(12)
)

type ConnectV1 struct {
	Credential string `json:"cre"`
	Namespace  string `json:"-"`
	Type       uint8  `json:"-"`
}

type ConnectResultV1 struct {
	AuthError string `json:"-"`
	Session   string `json:"s"`
	Key       string `json:"-"`
}

type EntityAlterV1 struct {
	Entities  []string `json:"args"`
	Namespace string   `json:"-"`
	Session   string   `json:"-"`
	Operation uint8    `json:"-"`
	Type      uint8    `json:"-"`
}

type MessagePushV1 struct {
	Msgs []MessageBody `json:"msg"`
}

type Subscription struct {
	Namespace string `json:"-"`
	Session   string `json:"s"`
	Group     string `json:"g"`
	Op        uint8  `json:"-"`
}
