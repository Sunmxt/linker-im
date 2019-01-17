package server

type GroupID [16]byte
type UserID [16]byte

type Message struct {
	Timestamp  uint64
	SequenceID uint32
	GroupID
	UserID
	Raw []byte
}
