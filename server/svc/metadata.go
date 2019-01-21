package svc

import (
    "encoding/binary"
)

type SerialiableEntity interface {
    Serialize() []byte 
    Unserialize(stream []byte) error
}

type NamespaceMetadata struct {
}

func (dat *NamespaceMetadata) Serialize() []byte {
    return make([]byte, 0)
}

func (dat *NamespaceMetadata) Unserialize([]byte) error {
    return nil
}

func NewDefaultNamespaceMetadata() *NamespaceMetadata {
    return &NamespaceMetadata{}
}

type GroupMetadata struct {
}

func (dat *GroupMetadata) Serialize() []byte {
    return make([]byte, 0)
}

func (dat *GroupMetadata) Unserialize([]byte) error {
    return nil
}

func NewDefaultGroupMetadata() *GroupMetadata {
    return &GroupMetadata{}
}
