package server

import (
	"errors"
	guuid "github.com/satori/go.uuid"
	"strings"
)

// Node ID
type NodeID guuid.UUID

var EMPTY_NODE_ID []byte = []byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0}

func NewNodeID() NodeID {
	return NodeID(guuid.NewV4())
}

func (n *NodeID) String() string {
	return strings.Replace(guuid.UUID(*n).String(), "-", "", -1)
}

func (n *NodeID) AsKey() string {
	return string(n[:])
}

func (n *NodeID) Assign(id *NodeID) {
	copy(n[:], id[:])
}

func (n *NodeID) FromString(raw string) error {
	return errors.New("Not implemented.")
}
