package proto

type ConnectV1 struct {
	Credentials []string `json:"cre"`
	Namespace   string   `json:"-"`
}

type ConnectResultV1 struct {
	AuthError string `json:"-"`
	Session   string `json:"s"`
	Key       string `json:"-"`
}

type EntityAlterV1 struct {
	Entities []string `json:"args"`
}

type MessagePushV1 struct {
	Msgs []MessageBody `json:"msg"`
}

const (
	OP_SUB_ADD    = uint8(0)
	OP_SUB_CANCEL = uint8(1)
)

const (
	ENTITY_NAMESPACE = uint8(1)
	ENTITY_USER      = uint8(2)
	ENTITY_GROUP     = uint8(3)

	ENTITY_ADD = uint8(1)
	ENTITY_DEL = uint8(2)
)

type Subscription struct {
	Namespace string `json:"-"`
	Session   string `json:"s"`
	Group     string `json:"g"`
	Op        uint8  `json:"-"`
}
