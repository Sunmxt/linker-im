package proto

import (
//guuid "github.com/satori/go.uuid"
//"strings"
)

//type ID [16]byte
//
//var EMPTY_ID []byte = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}
//
//func NewID() ID {
//	return ID(guuid.NewV4())
//}
//
//func (id ID) FromKey(key string) error {
//	return nil
//}
//
//func (id ID) String() {
//	return strings.Replace(guuid.UUID(*n).String(), "-", "", -1)
//}
//
//func (id ID) AsKey() string {
//	return string(id[:])
//}

type RawMessage struct {
	Namespace string
	Group     string
	User      string
	Raw       []byte
}

type MessageIdentifier struct {
	Timestamp  uint64
	SequenceID uint32
}

type Message struct {
	Identifier MessageIdentifier
	Content    RawMessage
}
