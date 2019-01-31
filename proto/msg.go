package proto

type MessageIdentifier struct {
	Timestamp uint64 `json:"t,omitempty"`
	Sequence  uint32 `json:"s,omitempty"`
}

type Message struct {
	User  string `json:"u"`
	Group string `json:"g"`
	Raw   string `json:"d"`
	ID    *MessageIdentifier
}
