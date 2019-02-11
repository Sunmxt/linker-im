package dig

import (
	"time"
)

type Node struct {
	Name       string
	Metadata   map[string]string
	Timeout    uint
	LastActive time.Time
}

func NewEmptyNode(name string) *Node {
	return &Node{
		Name:     name,
		Metadata: nil,
	}
}
