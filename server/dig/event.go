package dig

const (
	EVENT_SERVICE_FOUND         =       uint(1)
	//EVENT_SERVICE_LOST          =       uint(2)
    EVENT_NODE_LOST                 =       uint(3)
    EVENT_NODE_FOCUS                =       uint(4)
	EVENT_SVC_NODE_FOUND            =       uint(5)
	EVENT_SVC_NODE_LOST             =       uint(6)
	EVENT_NODE_METADATA_KEY_ADD     =       uint(7)
    EVENT_NODE_METADATA_KEY_DEL     =       uint(8)
    EVENT_NODE_METADATA_KEY_CHANGED =       uint(9)
)

type Notification struct {
    Event       uint
    Name        string
    *Node
}
