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

type ConnectV1 struct {
	Type       uint8  `json:"-"`
	Namespace  string `json:"-"`
	Credential string `json:"cre"`
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
	Msgs      []MessageBody `json:"msg"`
	Namespace string        `json:"-"`
	Session   string        `json:"-"`
}

type Subscription struct {
	Op        uint8  `json:"-"`
	Namespace string `json:"-"`
	Session   string `json:"s"`
	Group     string `json:"g"`
}
