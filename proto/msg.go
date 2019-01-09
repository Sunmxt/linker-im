package proto

type GroupID [16]byte
type UserID [16]byte

type RawMessage struct {
    Namespace   string
    GroupID
    UserID
    Raw []byte
}

type MessageIdentifier struct {
    Timestamp   uint64
    SequenceID  uint32
}

type Message struct {
    Identifier MessageIdentifier
    RawMessage
}
